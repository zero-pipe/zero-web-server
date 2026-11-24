package onvifapp

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	domainonvif "zero-web-server/internal/domain/onvif"
	applog "zero-web-server/pkg/log"

	onviflib "zero-web-server/3rdpart/onvif-go"
)

// EventSink 是平台事件落库入口（eventingest.Service.HandleMessage）。
//
// ONVIF 事件在进程内直接投递，不绕 MQTT：插件那条链路走 MQTT 是因为插件是
// 独立进程；ONVIF 订阅就在平台进程里，绕一圈只会多一个失败点。
type EventSink interface {
	HandleMessage(topic string, body []byte) error
}

// ONVIF 事件拉取节奏。
const (
	// eventPullTimeout 单次 PullMessages 的长轮询时长。太短会空转，太长会让
	// 停订阅时的退出变慢（拉取是在途请求）。20s 是常见的折中。
	eventPullTimeout = 20 * time.Second
	eventPullLimit   = 32
	// eventReconcile 是全量对齐周期：新加的设备建订阅、删掉的设备撤订阅。
	eventReconcile = 60 * time.Second
	// eventRetryDelay 订阅/拉取失败后的重试间隔。
	eventRetryDelay = 5 * time.Second
)

// eventSubLife 是向设备申请的订阅有效期；续订在 2/3 处触发。
// 用变量而非常量：CreatePullPointSubscription 要指针。
var eventSubLife = 300 * time.Second

// SetEventSink 注入事件落库入口。未注入时不做事件订阅（保持原有行为）。
func (s *Service) SetEventSink(sink EventSink) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eventSink = sink
}

// StartEventSync 启动 ONVIF 事件订阅的常驻同步。
//
// 用"周期对齐"而不是在增删设备的每个入口挂钩子：设备增删、改名、改密码、
// 离线恢复都只是让下一轮对齐发现差异，不会漏掉某条代码路径。
func (s *Service) StartEventSync(ctx context.Context) {
	if s.eventSinkFor() == nil {
		applog.Warn("[onvif] 未注入事件入口，跳过事件订阅")
		return
	}
	go func() {
		s.reconcileEventSubs(ctx)
		ticker := time.NewTicker(eventReconcile)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				s.stopAllEventSubs()
				return
			case <-ticker.C:
				s.reconcileEventSubs(ctx)
			}
		}
	}()
}

func (s *Service) eventSinkFor() EventSink {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.eventSink
}

// reconcileEventSubs 把订阅集合对齐到当前启用的设备集合。
func (s *Service) reconcileEventSubs(ctx context.Context) {
	devices, err := s.listAllDevices(ctx)
	if err != nil {
		applog.Warnf("[onvif] 事件订阅对齐：读取设备列表失败: %v", err)
		return
	}
	want := make(map[int64]*domainonvif.Device, len(devices))
	for _, d := range devices {
		if d == nil || strings.TrimSpace(d.DeviceURI) == "" {
			continue
		}
		want[d.ID] = d
	}

	// 先撤掉已经不存在 / 已离线的设备订阅。
	s.evMu.Lock()
	stale := make([]int64, 0, len(s.evSubs))
	for id, cancel := range s.evSubs {
		if _, ok := want[id]; ok {
			continue
		}
		cancel()
		delete(s.evSubs, id)
		stale = append(stale, id)
	}
	s.evMu.Unlock()
	for _, id := range stale {
		applog.Infof("[onvif] 设备 %d 已移除，撤销事件订阅", id)
	}

	// 再补齐新设备的订阅。已订阅的设备不重复建（哪怕这轮它离线了，
	// 由拉取循环自己重试，避免把健康订阅拆了重建）。
	//
	// 这里不看 OnLine 标记：ONVIF 的在线状态只在手动探测时刷新，可能长期是旧值。
	// 真正可用与否由订阅/拉取自己判断，失败的设备按退避重试，不会影响别的设备。
	for id, dev := range want {
		s.evMu.Lock()
		_, exists := s.evSubs[id]
		s.evMu.Unlock()
		if exists {
			continue
		}
		subCtx, cancel := context.WithCancel(context.Background())
		s.evMu.Lock()
		s.evSubs[id] = cancel
		s.evMu.Unlock()
		go s.runEventPump(subCtx, dev, cancel)
		applog.Infof("[onvif] 设备 %s（%d）开始事件订阅", dev.Name, id)
	}
}

// runEventPump 维护单台设备的 PullPoint 订阅与消息拉取。
func (s *Service) runEventPump(ctx context.Context, dev *domainonvif.Device, release context.CancelFunc) {
	defer release()
	sink := s.eventSinkFor()
	if sink == nil {
		return
	}
	subFails := 0
	for {
		if ctx.Err() != nil {
			return
		}
		client, err := s.newEventClient(ctx, dev)
		if err != nil {
			subFails++
			s.logSubFailure(dev, subFails, err)
			if !sleepCtx(ctx, eventRetryDelay*6) {
				return
			}
			continue
		}
		sub, err := client.CreatePullPointSubscription(ctx, "", &eventSubLife, "")
		if err != nil {
			subFails++
			s.logSubFailure(dev, subFails, err)
			if !sleepCtx(ctx, eventRetryDelay*6) {
				return
			}
			continue
		}
		if subFails > 0 {
			applog.Infof("[onvif] 设备 %s 事件订阅已恢复", dev.Name)
			subFails = 0
		}
		s.pullLoop(ctx, dev, client, sub, sink)
	}
}

// logSubFailure 只在第一次和每 20 次打 Warn：一台长期离线的设备
// 不该每 30 秒刷一条告警把日志淹掉。
func (s *Service) logSubFailure(dev *domainonvif.Device, n int, err error) {
	if n == 1 || n%20 == 0 {
		applog.Warnf("[onvif] 设备 %s 订阅事件失败（第 %d 次，设备可能离线或不支持 PullPoint）: %v",
			dev.Name, n, err)
	}
}

// pullLoop 在订阅有效期内持续拉取，到期前续订；出错就重订阅。
func (s *Service) pullLoop(ctx context.Context, dev *domainonvif.Device, client *onviflib.Client,
	sub *onviflib.PullPointSubscription, sink EventSink) {
	ref := sub.SubscriptionReference
	renewAt := time.Now().Add(eventSubLife * 2 / 3)
	if !sub.TerminationTime.IsZero() {
		// 设备给的终止时间才是准的，按它算续订点。
		if d := time.Until(sub.TerminationTime); d > 0 {
			renewAt = time.Now().Add(d * 2 / 3)
		}
	}
	failures := 0
	for {
		if ctx.Err() != nil {
			// 退出前主动退订，别把订阅留在设备上占配额。
			_ = client.Unsubscribe(context.WithoutCancel(ctx), ref)
			return
		}
		if time.Now().After(renewAt) {
			cur, term, err := client.RenewSubscription(ctx, ref, eventSubLife)
			if err != nil {
				applog.Warnf("[onvif] 设备 %s 续订失败，重建订阅: %v", dev.Name, err)
				return
			}
			_ = cur
			next := time.Now().Add(eventSubLife * 2 / 3)
			if !term.IsZero() {
				if d := time.Until(term); d > 0 {
					next = time.Now().Add(d * 2 / 3)
				}
			}
			renewAt = next
		}

		msgs, err := client.PullMessages(ctx, ref, eventPullTimeout, eventPullLimit)
		if err != nil {
			if ctx.Err() != nil {
				continue
			}
			// 长轮询超时是常态，多数设备以 Fault 形式返回；不算失败。
			failures++
			if failures == 1 || failures%10 == 0 {
				applog.Debugf("[onvif] 设备 %s 拉取事件失败（第 %d 次）: %v", dev.Name, failures, err)
			}
			if failures >= 3 {
				// 连续失败说明订阅已失效（设备重启 / 会话被踢），重建。
				applog.Infof("[onvif] 设备 %s 拉取连续失败，重建订阅", dev.Name)
				return
			}
			if !sleepCtx(ctx, eventRetryDelay) {
				return
			}
			continue
		}
		failures = 0
		if len(msgs) == 0 {
			continue
		}
		for _, m := range msgs {
			s.dispatchNotification(dev, m, sink)
		}
	}
}

// dispatchNotification 把一条 ONVIF 通知翻译成平台事件并投递。
func (s *Service) dispatchNotification(dev *domainonvif.Device, m onviflib.NotificationMessage, sink EventSink) {
	topic := strings.TrimSpace(m.Topic)
	if topic == "" {
		return
	}
	// 状态型通知（Properties/Changed）只在"进入报警态"时投递，
	// 否则一次移动侦测会连着刷 true/false 两条，事件列表全是噪声。
	if !notificationActive(m) {
		return
	}
	code := mapONVIFTopic(topic)
	if code == "" {
		applog.Debugf("[onvif] 设备 %s 未映射的事件主题: %s", dev.Name, topic)
		return
	}
	occurred := m.Message.UtcTime
	if occurred.IsZero() {
		occurred = time.Now()
	}
	payload := map[string]any{
		"schema":             "1",
		"eventId":            onvifEventID(dev, topic, occurred),
		"deviceInternalCode": dev.InternalCode,
		"deviceUid":          firstNonEmpty(dev.SerialNumber, dev.DeviceURI, dev.IP),
		"eventType":          code,
		"occurredAt":         occurred.UnixMilli(),
		"receivedAt":         time.Now().UnixMilli(),
	}
	if ch := s.matchChannel(dev, m); ch != nil {
		payload["channelId"] = ch.ProfileToken
		payload["channelInternalCode"] = ch.InternalCode
	}
	raw, _ := json.Marshal(map[string]any{
		"topic":   topic,
		"source":  simpleItems(m.Message.Source),
		"data":    simpleItems(m.Message.Data),
		"key":     simpleItems(m.Message.Key),
		"utcTime": occurred.Format(time.RFC3339Nano),
	})
	payload["raw"] = json.RawMessage(raw)
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	if err := sink.HandleMessage("onvif/event", body); err != nil {
		applog.Debugf("[onvif] 设备 %s 事件落库失败: %v", dev.Name, err)
	}
}

// matchChannel 用通知里的视频源标识找平台通道。
//
// ONVIF 通知带的是 VideoSourceToken / VideoSourceConfigurationToken，
// 与通道上保存的 VideoSource 对得上；对不上时退回设备的第一个通道——
// 单通道枪机是绝大多数场景，宁可归到唯一通道也不要丢事件。
func (s *Service) matchChannel(dev *domainonvif.Device, m onviflib.NotificationMessage) *domainonvif.Channel {
	channels, err := s.channels.ListByDeviceID(context.Background(), dev.ID)
	if err != nil || len(channels) == 0 {
		return nil
	}
	tokens := map[string]bool{}
	for _, it := range m.Message.Source {
		v := strings.TrimSpace(it.Value)
		if v != "" {
			tokens[v] = true
		}
	}
	for _, ch := range channels {
		if ch.VideoSource != "" && tokens[ch.VideoSource] {
			return ch
		}
	}
	if len(channels) == 1 {
		return channels[0]
	}
	return nil
}

// newEventClient 复用秒级超时的工厂客户端；事件端点由库自行发现。
func (s *Service) newEventClient(ctx context.Context, dev *domainonvif.Device) (*onviflib.Client, error) {
	client, err := s.factory.Initialize(ctx, dev.DeviceURI, dev.Username, dev.Password)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("ONVIF 客户端不可用")
	}
	return client, nil
}

func (s *Service) listAllDevices(ctx context.Context) ([]*domainonvif.Device, error) {
	const pageSize = 200
	var all []*domainonvif.Device
	for page := 1; ; page++ {
		list, total, err := s.devices.List(ctx, page, pageSize, "")
		if err != nil {
			return nil, err
		}
		all = append(all, list...)
		if len(list) < pageSize || int64(len(all)) >= total {
			return all, nil
		}
	}
}

func (s *Service) stopAllEventSubs() {
	s.evMu.Lock()
	cancels := make([]context.CancelFunc, 0, len(s.evSubs))
	for id, cancel := range s.evSubs {
		cancels = append(cancels, cancel)
		delete(s.evSubs, id)
	}
	s.evMu.Unlock()
	for _, c := range cancels {
		c()
	}
}

// ---- 通知解析 ----

// notificationActive 判断通知是否处于"触发中"。
//
// 视频类事件有明确三态：true/false（PropertyOperation）、或 Data 里的
// State/LogicalState/IsMotion。设备不发状态的时候按"触发"处理。
func notificationActive(m onviflib.NotificationMessage) bool {
	for _, it := range m.Message.Data {
		switch strings.ToLower(it.Name) {
		case "state", "logicalstate", "ismotion", "active", "alarmstate", "value":
			return truthy(it.Value)
		}
	}
	for _, it := range m.Message.Source {
		if strings.EqualFold(it.Name, "state") {
			return truthy(it.Value)
		}
	}
	return true
}

func truthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "false", "0", "inactive", "off", "no", "disable", "disabled":
		return false
	}
	return true
}

// onvifTopicRules 把 ONVIF 主题映射到平台规范事件码。
//
// 顺序有讲究：先匹配最具体的（越界/入侵/徘徊），否则 "Motion" 会把
// RuleEngine 下的复合规则全吃掉。子串一律小写比较。
var onvifTopicRules = []struct {
	keywords []string
	code     string
}{
	{[]string{"linecrossing", "linedetector", "linecrossed"}, "line_crossing"},
	{[]string{"fielddetector", "intrusion", "intruderdetector"}, "intrusion"},
	{[]string{"loitering", "loiterdetector"}, "loitering"},
	{[]string{"tamper", "defocus"}, "tamper"},
	{[]string{"scenechange", "globalscenechange", "imagetooblurry"}, "scene_change"},
	{[]string{"videoloss", "videosignalloss", "signalloss"}, "video_loss"},
	{[]string{"audioclassified", "audioabnormal", "audiodetect", "audio"}, "audio_abnormal"},
	{[]string{"facedetect", "faceanalysis", "face"}, "face"},
	{[]string{"licensplate", "licenseplate", "lpr", "vehicleplate"}, "plate"},
	{[]string{"motion", "cellmotiondetector", "movement"}, "motion"},
	{[]string{"digitalinput", "alarminput", "inputsourcelocal", "externalalarm"}, "alarm_input"},
	{[]string{"ptz", "ptzmoving", "ptzmoved"}, "ptz"},
	{[]string{"record", "storage", "disks", "recording"}, "record_abnormal"},
	{[]string{"mediaconfiguration", "configurationchanged", "config"}, "config_change"},
}

// mapONVIFTopic 返回平台事件码；无法映射时返回空串（由调用方丢弃）。
//
// 注意不要把未识别主题一律塞进 other：ONVIF 的 tns1:监控类主题量很大，
// 全量入 other 会把事件列表淹掉。丢弃并打 debug 日志更好排查。
func mapONVIFTopic(topic string) string {
	low := strings.ToLower(topic)
	for _, rule := range onvifTopicRules {
		for _, kw := range rule.keywords {
			if strings.Contains(low, kw) {
				return rule.code
			}
		}
	}
	return ""
}

func onvifEventID(dev *domainonvif.Device, topic string, at time.Time) string {
	h := sha1.Sum([]byte(fmt.Sprintf("%s|%s|%d", dev.InternalCode, topic, at.UnixNano())))
	return "onvif-" + hex.EncodeToString(h[:10])
}

func simpleItems(items []onviflib.SimpleItem) map[string]string {
	if len(items) == 0 {
		return nil
	}
	out := make(map[string]string, len(items))
	for _, it := range items {
		out[it.Name] = it.Value
	}
	return out
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
