//go:build hik && cgo

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type armMQTTConfig struct {
	Broker     string `json:"broker"`
	ClientID   string `json:"clientId"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	Topic      string `json:"topic"`
	QoS        int    `json:"qos"`
	KeepAlive  int    `json:"keepAlive"`
	TLSEnabled bool   `json:"tls"`
	CACert     string `json:"caCert"`
	ClientCert string `json:"clientCert"`
	ClientKey  string `json:"clientKey"`
}

type armObjectStoreConfig struct {
	Provider   string `json:"provider"`
	Endpoint   string `json:"endpoint"`
	Region     string `json:"region"`
	Bucket     string `json:"bucket"`
	AccessKey  string `json:"accessKey"`
	SecretKey  string `json:"secretKey"`
	UseSSL     bool   `json:"useSSL"`
	PathStyle  bool   `json:"pathStyle"`
	PublicBase string `json:"publicBase"`
}

type armRequest struct {
	DeviceUID          string               `json:"device_uid"`
	DeviceInternalCode string               `json:"device_internal_code"`
	EventTypes         []string             `json:"event_types"`
	MQTT               armMQTTConfig        `json:"mqtt"`
	ObjectStore        armObjectStoreConfig `json:"object_store"`
}

type pipeline struct {
	ctx    context.Context
	cancel context.CancelFunc

	mqtt  mqtt.Client
	topic string
	qos   byte

	store      *minio.Client
	bucket     string
	publicBase string

	deviceUID     string
	internal      string
	ip            string
	user          string
	pass          string
	isapiPort     int
	isapiHTTPS    bool
	isapiInsecure bool
	eventTypes    map[string]bool
}

func startPipeline(parent context.Context, s *session, cfg armRequest) (*pipeline, error) {
	endpoint := strings.TrimSpace(cfg.ObjectStore.Endpoint)
	endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")
	endpoint = strings.TrimRight(endpoint, "/")
	if endpoint == "" {
		return nil, fmt.Errorf("object_store.endpoint 不能为空")
	}
	if strings.TrimSpace(cfg.ObjectStore.Bucket) == "" {
		return nil, fmt.Errorf("object_store.bucket 不能为空")
	}
	lookup := minio.BucketLookupAuto
	if cfg.ObjectStore.PathStyle {
		lookup = minio.BucketLookupPath
	}
	store, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.ObjectStore.AccessKey, cfg.ObjectStore.SecretKey, ""),
		Secure:       cfg.ObjectStore.UseSSL,
		Region:       cfg.ObjectStore.Region,
		BucketLookup: lookup,
	})
	if err != nil {
		return nil, fmt.Errorf("初始化对象存储失败: %w", err)
	}
	checkCtx, cancel := context.WithTimeout(parent, 6*time.Second)
	ok, err := store.BucketExists(checkCtx, cfg.ObjectStore.Bucket)
	cancel()
	if err != nil {
		return nil, fmt.Errorf("对象存储不可用: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("对象存储 bucket 不存在: %s", cfg.ObjectStore.Bucket)
	}

	if strings.TrimSpace(cfg.MQTT.Broker) == "" || strings.TrimSpace(cfg.MQTT.Topic) == "" {
		return nil, fmt.Errorf("mqtt.broker/topic 不能为空")
	}
	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.MQTT.Broker)
	opts.SetClientID(cfg.MQTT.ClientID)
	if cfg.MQTT.Username != "" {
		opts.SetUsername(cfg.MQTT.Username)
	}
	if cfg.MQTT.Password != "" {
		opts.SetPassword(cfg.MQTT.Password)
	}
	keepAlive := cfg.MQTT.KeepAlive
	if keepAlive <= 0 {
		keepAlive = 30
	}
	opts.SetKeepAlive(time.Duration(keepAlive) * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetConnectTimeout(6 * time.Second)
	opts.SetConnectRetry(false)
	if cfg.MQTT.TLSEnabled {
		tlsCfg, err := pluginTLS(cfg.MQTT.CACert, cfg.MQTT.ClientCert, cfg.MQTT.ClientKey)
		if err != nil {
			return nil, err
		}
		opts.SetTLSConfig(tlsCfg)
	}
	cli := mqtt.NewClient(opts)
	token := cli.Connect()
	if !token.WaitTimeout(8 * time.Second) {
		return nil, fmt.Errorf("MQTT 连接超时")
	}
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("MQTT 连接失败: %w", err)
	}

	isapi := newISAPIClient(s.IP, hikConfig.isapiPort, hikConfig.isapiHTTPS, hikConfig.isapiInsecure, s.User, s.Pass)
	probeCtx, probeCancel := context.WithTimeout(parent, 6*time.Second)
	resp, err := isapi.get(probeCtx, "/ISAPI/System/deviceInfo")
	probeCancel()
	if err != nil {
		cli.Disconnect(250)
		return nil, fmt.Errorf("ISAPI 连接失败: %w", err)
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		cli.Disconnect(250)
		return nil, fmt.Errorf("ISAPI 认证失败: HTTP %d", resp.StatusCode)
	}

	types := map[string]bool{}
	for _, t := range cfg.EventTypes {
		types[strings.ToLower(strings.TrimSpace(t))] = true
	}
	ctx, cancelFn := context.WithCancel(parent)
	p := &pipeline{
		ctx: ctx, cancel: cancelFn, mqtt: cli, topic: cfg.MQTT.Topic, qos: byte(cfg.MQTT.QoS),
		store: store, bucket: cfg.ObjectStore.Bucket, publicBase: strings.TrimRight(cfg.ObjectStore.PublicBase, "/"),
		deviceUID: cfg.DeviceUID, internal: cfg.DeviceInternalCode, ip: s.IP, user: s.User, pass: s.Pass,
		isapiPort: hikConfig.isapiPort, isapiHTTPS: hikConfig.isapiHTTPS, isapiInsecure: hikConfig.isapiInsecure,
		eventTypes: types,
	}
	go p.run()
	return p, nil
}

func (p *pipeline) stop() {
	if p == nil {
		return
	}
	p.cancel()
	if p.mqtt != nil && p.mqtt.IsConnectionOpen() {
		p.mqtt.Disconnect(250)
	}
}

func (p *pipeline) run() {
	backoff := 3 * time.Second
	for {
		if p.ctx.Err() != nil {
			return
		}
		if err := p.streamOnce(); err != nil {
			log.Printf("hik alertStream device=%s err=%v (retry in %s)", p.deviceUID, err, backoff)
			select {
			case <-p.ctx.Done():
				return
			case <-time.After(backoff):
			}
			continue
		}
	}
}

func (p *pipeline) streamOnce() error {
	isapi := newISAPIClient(p.ip, p.isapiPort, p.isapiHTTPS, p.isapiInsecure, p.user, p.pass)
	resp, err := isapi.get(p.ctx, "/ISAPI/Event/notification/alertStream")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("alertStream HTTP %d", resp.StatusCode)
	}
	mediaType, params, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if strings.HasPrefix(mediaType, "multipart/") && params["boundary"] != "" {
		mr := multipart.NewReader(resp.Body, params["boundary"])
		for {
			part, err := mr.NextPart()
			if err != nil {
				return err
			}
			data, err := io.ReadAll(io.LimitReader(part, 1<<20))
			part.Close()
			if err != nil {
				return err
			}
			p.handleAlert(data)
		}
	}
	// Some firmware returns a stream of XML documents instead of multipart.
	dec := xml.NewDecoder(resp.Body)
	for {
		var alert eventAlert
		if err := dec.Decode(&alert); err != nil {
			return err
		}
		if alert.EventType != "" {
			p.handleAlert([]byte(fmt.Sprintf(`<EventNotificationAlert><eventType>%s</eventType><eventState>%s</eventState><channelID>%s</channelID><dateTime>%s</dateTime><eventDescription>%s</eventDescription></EventNotificationAlert>`,
				escapeXML(alert.EventType), escapeXML(alert.EventState), escapeXML(alert.ChannelID), escapeXML(alert.DateTime), escapeXML(alert.EventDescription))))
		}
	}
}

type eventAlert struct {
	EventType        string `xml:"eventType" json:"eventType"`
	EventState       string `xml:"eventState" json:"eventState"`
	ChannelID        string `xml:"channelID" json:"channelID"`
	DateTime         string `xml:"dateTime" json:"dateTime"`
	EventDescription string `xml:"eventDescription" json:"eventDescription"`
}

func (p *pipeline) handleAlert(data []byte) {
	var alert eventAlert
	if err := xml.Unmarshal(data, &alert); err != nil || alert.EventType == "" {
		// Try JSON payloads (some 7.x firmware).
		if jerr := json.Unmarshal(data, &alert); jerr != nil || alert.EventType == "" {
			return
		}
	}
	if strings.EqualFold(alert.EventState, "inactive") {
		return
	}
	eventType := normalizeEventType(alert.EventType)
	if len(p.eventTypes) > 0 && !p.eventTypes["all"] && !p.eventTypes[eventType] {
		return
	}
	channel := strings.TrimSpace(alert.ChannelID)
	if channel == "" {
		channel = "1"
	}
	occurred := parseHikTime(alert.DateTime)
	eventID := fmt.Sprintf("%s-%d-%s", sanitizeKey(p.internal), time.Now().UnixNano(), randomHex(4))

	image := p.captureAndUpload(channel, eventID)
	event := map[string]any{
		"schema":             "zws.event.v1",
		"eventId":            eventID,
		"deviceInternalCode": p.internal,
		"deviceUid":          p.deviceUID,
		"channelId":          channel,
		"eventType":          eventType,
		"severity":           "warning",
		"occurredAt":         occurred,
		"receivedAt":         time.Now().UnixMilli(),
		"raw": map[string]any{
			"vendor": "hikvision", "eventType": alert.EventType, "eventState": alert.EventState,
			"eventDescription": alert.EventDescription, "dateTime": alert.DateTime,
		},
	}
	if image != nil {
		event["image"] = image
	}
	payload, _ := json.Marshal(event)
	token := p.mqtt.Publish(p.topic, p.qos, false, payload)
	if !token.WaitTimeout(5*time.Second) || token.Error() != nil {
		log.Printf("hik event publish failed device=%s event=%s err=%v", p.deviceUID, eventID, token.Error())
		return
	}
	log.Printf("hik event published device=%s internal=%s type=%s channel=%s event=%s image=%v",
		p.deviceUID, p.internal, eventType, channel, eventID, image != nil)
}

func (p *pipeline) captureAndUpload(channel, eventID string) map[string]any {
	path := fmt.Sprintf("/ISAPI/Streaming/channels/%d/picture", channelNumber(channel))
	ctx, cancel := context.WithTimeout(p.ctx, 6*time.Second)
	defer cancel()
	isapi := newISAPIClient(p.ip, p.isapiPort, p.isapiHTTPS, p.isapiInsecure, p.user, p.pass)
	resp, err := isapi.get(ctx, path)
	if err != nil {
		log.Printf("hik snapshot failed device=%s channel=%s err=%v", p.deviceUID, channel, err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("hik snapshot HTTP %d device=%s channel=%s", resp.StatusCode, p.deviceUID, channel)
		return nil
	}
	img, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil || len(img) == 0 {
		log.Printf("hik snapshot empty device=%s channel=%s", p.deviceUID, channel)
		return nil
	}
	now := time.Now()
	key := fmt.Sprintf("events/%s/%04d/%02d/%02d/%s.jpg", sanitizeKey(p.internal), now.Year(), now.Month(), now.Day(), eventID)
	putCtx, putCancel := context.WithTimeout(p.ctx, 10*time.Second)
	defer putCancel()
	if _, err := p.store.PutObject(putCtx, p.bucket, key, bytes.NewReader(img), int64(len(img)), minio.PutObjectOptions{ContentType: "image/jpeg"}); err != nil {
		log.Printf("hik event image upload failed device=%s key=%s err=%v", p.deviceUID, key, err)
		return nil
	}
	image := map[string]any{"key": key}
	if p.publicBase != "" {
		image["url"] = p.publicBase + "/" + key
	} else if u, err := p.store.PresignedGetObject(putCtx, p.bucket, key, 24*time.Hour, url.Values{}); err == nil {
		image["url"] = u.String()
		image["expiresAt"] = time.Now().Add(24 * time.Hour).UnixMilli()
	}
	return image
}

func channelNumber(channel string) int {
	n, err := strconv.Atoi(channel)
	if err != nil || n <= 0 {
		return 101
	}
	return n*100 + 1
}

func normalizeEventType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "vmd", "motion", "motiondetection":
		return "motion"
	case "linedetection", "linecrossing", "line_crossing":
		return "line_crossing"
	case "fielddetection", "intrusion", "regionentrance", "regionexiting":
		return "intrusion"
	case "videoloss", "video_loss":
		return "video_loss"
	case "tamper", "shelteralarm":
		return "tamper"
	case "facedetection", "face":
		return "face"
	}
	return strings.ToLower(strings.TrimSpace(raw))
}

func parseHikTime(v string) int64 {
	for _, layout := range []string{
		time.RFC3339, "2006-01-02T15:04:05-07:00", "2006-01-02T15:04:05", "2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, v); err == nil {
			return t.UnixMilli()
		}
	}
	return time.Now().UnixMilli()
}

func sanitizeKey(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "unknown"
	}
	return b.String()
}

func escapeXML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")
	return r.Replace(s)
}

func pluginTLS(ca, cert, key string) (*tls.Config, error) {
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if strings.TrimSpace(ca) != "" {
		pool := x509.NewCertPool()
		pem, err := os.ReadFile(ca)
		if err != nil {
			return nil, fmt.Errorf("read mqtt ca: %w", err)
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("invalid mqtt ca")
		}
		cfg.RootCAs = pool
	}
	if cert != "" && key != "" {
		pair, err := tls.LoadX509KeyPair(cert, key)
		if err != nil {
			return nil, fmt.Errorf("load mqtt client cert: %w", err)
		}
		cfg.Certificates = []tls.Certificate{pair}
	}
	return cfg, nil
}
