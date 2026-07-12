package sipinfra

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	domainchannel "zero-web-kit/internal/domain/channel"
	domaindevice "zero-web-kit/internal/domain/device"
	domainptz "zero-web-kit/internal/domain/ptz"
	domainrecord "zero-web-kit/internal/domain/record"
	"zero-web-kit/internal/infrastructure/config"
	redisinfra "zero-web-kit/internal/infrastructure/redis"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

type DeviceService interface {
	GetByDeviceID(deviceID string) (*domaindevice.Device, error)
	Online(device *domaindevice.Device) error
	Offline(device *domaindevice.Device) error
	SaveRegister(device *domaindevice.Device) (*domaindevice.Device, error)
	HandleKeepalive(deviceID, ip string, port int) error
	HandleCatalog(deviceID string, items []CatalogItem) error
	HandleDeviceInfo(deviceID, name, manufacturer, model, firmware string) error
	OnDeviceOnline(device *domaindevice.Device)
}

type AlarmHandler interface {
	HandleNotify(deviceID, channelGBID string, alarm *AlarmNotify) error
}

type PositionHandler interface {
	HandleNotify(deviceID, channelGBID string, pos *MobilePositionNotify) error
}

type Server struct {
	cfg                 config.SIPConfig
	localIP             string
	serverID            string
	password            string
	requirePreRegister  bool
	ua                  *sipgo.UserAgent
	srv                 *sipgo.Server
	client              *sipgo.Client
	deviceSvc           DeviceService
	alarmHandler        AlarmHandler
	positionHandler     PositionHandler
	redis               *redisinfra.Client
	challenges          sync.Map
	sn                  atomic.Int64
	recordMgr           *RecordManager
	presetMgr           *PresetManager
	inviteMgr           *InviteManager
	infoCSeq            atomic.Int64
}

// SetRequirePreRegister 未预添加设备是否拒绝 REGISTER
func (s *Server) SetRequirePreRegister(v bool) {
	s.requirePreRegister = v
}

func NewServer(cfg config.SIPConfig, serverID, password string, deviceSvc DeviceService, redis *redisinfra.Client) (*Server, error) {
	localIP := strings.TrimSpace(cfg.IP)
	if localIP == "" || localIP == "0.0.0.0" || localIP == "127.0.0.1" {
		localIP = guessLocalIP()
	}
	uaName := strings.TrimSpace(cfg.ID)
	if uaName == "" {
		uaName = "zero-web-kit"
	}
	uaOpts := []sipgo.UserAgentOption{sipgo.WithUserAgent(uaName)}
	if localIP != "" {
		uaOpts = append(uaOpts, sipgo.WithUserAgentHostname(localIP))
	}
	ua, err := sipgo.NewUA(uaOpts...)
	if err != nil {
		return nil, err
	}
	srv, err := sipgo.NewServer(ua)
	if err != nil {
		return nil, err
	}
	clientOpts := []sipgo.ClientOption{}
	if localIP != "" && cfg.Port > 0 {
		clientOpts = append(clientOpts,
			sipgo.WithClientHostname(localIP),
			sipgo.WithClientPort(cfg.Port),
			sipgo.WithClientNAT(),
		)
	}
	client, err := sipgo.NewClient(ua, clientOpts...)
	if err != nil {
		return nil, err
	}
	s := &Server{
		cfg:       cfg,
		localIP:   localIP,
		serverID:  serverID,
		password:  password,
		ua:        ua,
		srv:       srv,
		client:    client,
		deviceSvc: deviceSvc,
		redis:     redis,
		recordMgr: NewRecordManager(),
		presetMgr: NewPresetManager(),
		inviteMgr: NewInviteManager(),
	}
	s.registerHandlers()
	if cfg.Port <= 0 || strings.TrimSpace(cfg.ID) == "" {
		log.Printf("[GB28181 sip] 未配置国标 SIP（请到「系统管理 → 国标配置」填写并保存）")
	} else if localIP == "" {
		log.Printf("[GB28181 sip] WARNING: sip.ip 未配置且自动探测失败，国标 INVITE 可能超时")
	} else {
		log.Printf("[GB28181 sip] localIP=%s (Contact/Via)", localIP)
	}
	return s, nil
}

func (s *Server) SetAlarmHandler(h AlarmHandler)       { s.alarmHandler = h }
func (s *Server) SetPositionHandler(h PositionHandler) { s.positionHandler = h }
func (s *Server) SetLocalIP(ip string) {
	ip = strings.TrimSpace(ip)
	if ip == "" || ip == "0.0.0.0" || ip == "127.0.0.1" {
		return
	}
	// 仅在尚未配置时用媒体节点 IP 兜底；已有 sip.ip 不覆盖
	if s.localIP == "" {
		s.localIP = ip
		log.Printf("[GB28181 sip] localIP set from media node: %s", ip)
	}
}

// ApplyConfig 热更新国标身份/密码/域/可达 IP。监听端口变更需重启进程。
func (s *Server) ApplyConfig(cfg config.SIPConfig) {
	s.cfg = cfg
	s.password = cfg.Password
	ip := strings.TrimSpace(cfg.IP)
	if ip != "" && ip != "0.0.0.0" && ip != "127.0.0.1" {
		s.localIP = ip
	}
	log.Printf("[GB28181 sip] config applied id=%s domain=%s ip=%s port=%d (listen port change needs restart)",
		cfg.ID, cfg.Domain, s.localIP, cfg.Port)
}

func (s *Server) Config() config.SIPConfig { return s.cfg }
func (s *Server) LocalIP() string          { return s.localIP }
func (s *Server) Domain() string           { return s.cfg.Domain }

// GuessLocalIP 取第一块非回环 IPv4，作为 sip.ip 未配置时的兜底。
func GuessLocalIP() string {
	return guessLocalIP()
}

// guessLocalIP 取第一块非回环 IPv4，作为 sip.ip 未配置时的兜底。
func guessLocalIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue
			}
			return ip.String()
		}
	}
	return ""
}

// detectLocalIPForRemote 通过 UDP dial 探测访问远端时本机选用的网卡 IP。
func detectLocalIPForRemote(remoteIP string) string {
	remoteIP = strings.TrimSpace(remoteIP)
	if remoteIP == "" {
		return ""
	}
	conn, err := net.DialTimeout("udp", net.JoinHostPort(remoteIP, "9"), time.Second)
	if err != nil {
		return ""
	}
	defer conn.Close()
	if ua, ok := conn.LocalAddr().(*net.UDPAddr); ok && ua.IP != nil {
		if v4 := ua.IP.To4(); v4 != nil {
			return v4.String()
		}
	}
	return ""
}

func (s *Server) RecordManager() *RecordManager { return s.recordMgr }
func (s *Server) PresetManager() *PresetManager { return s.presetMgr }

func (s *Server) SendRecordInfoQuery(device *domaindevice.Device, channelID, startTime, endTime string) (string, <-chan *domainrecord.RecordInfo) {
	sn := strconv.FormatInt(s.sn.Add(1), 10)
	ch := s.recordMgr.Register(sn)
	body := BuildRecordInfoQuery(channelID, sn, startTime, endTime)
	_ = s.sendMessage(device, body)
	return sn, ch
}

func (s *Server) CancelRecordQuery(sn string) { s.recordMgr.Cancel(sn) }

func (s *Server) SendPresetQuery(device *domaindevice.Device, channelID string) (string, <-chan []domainptz.Preset) {
	sn := strconv.FormatInt(s.sn.Add(1), 10)
	ch := s.presetMgr.Register(sn)
	body := BuildPresetQuery(channelID, sn)
	_ = s.sendMessage(device, body)
	return sn, ch
}

func (s *Server) CancelPresetQuery(sn string) { s.presetMgr.Cancel(sn) }

// SendFrontEndCmd sends DeviceControl PTZCmd (front-end command).
func (s *Server) SendFrontEndCmd(device *domaindevice.Device, channelID string, cmdCode, parameter1, parameter2, combineCode2 int) error {
	sn := strconv.FormatInt(s.sn.Add(1), 10)
	cmd := FrontEndCmdString(cmdCode, parameter1, parameter2, combineCode2)
	body := BuildDeviceControlFrontEnd(channelID, sn, cmd)
	return s.sendMessage(device, body)
}

func (s *Server) registerHandlers() {
	s.srv.OnRegister(s.handleRegister)
	s.srv.OnMessage(s.handleMessage)
	s.srv.OnBye(s.handleBye)
}

func (s *Server) handleBye(req *sip.Request, tx sip.ServerTransaction) {
	from := extractSIPUser(req)
	log.Printf("[GB28181 sip] BYE from=%s source=%s", from, req.Source())
	s.respond(tx, req, 200, "OK", nil)
}

func (s *Server) Start(ctx context.Context) error {
	if s.cfg.Port <= 0 {
		log.Printf("GB28181 SIP 未监听：端口未配置，请到「系统管理 → 国标配置」保存后重启服务")
		return nil
	}
	addr := fmt.Sprintf("0.0.0.0:%d", s.cfg.Port)
	go func() {
		if err := s.srv.ListenAndServe(ctx, "udp", addr); err != nil && ctx.Err() == nil {
			log.Printf("sip udp server error: %v", err)
		}
	}()
	go func() {
		if err := s.srv.ListenAndServe(ctx, "tcp", addr); err != nil && ctx.Err() == nil {
			log.Printf("sip tcp server error: %v", err)
		}
	}()
	log.Printf("GB28181 SIP listening on %s (TCP+UDP)", addr)
	return nil
}

func (s *Server) handleRegister(req *sip.Request, tx sip.ServerTransaction) {
	deviceID := extractSIPUser(req)
	if deviceID == "" {
		s.respond(tx, req, 400, "Bad Request", nil)
		return
	}

	expires := parseExpires(req)
	ip, port := extractSourceAddr(req)

	if expires == 0 {
		s.handleUnregister(req, tx, deviceID)
		return
	}

	device, _ := s.deviceSvc.GetByDeviceID(deviceID)
	if device == nil && s.requirePreRegister {
		log.Printf("GB28181 register rejected (not pre-registered): id=%s ip=%s:%d", deviceID, ip, port)
		s.respond(tx, req, 403, "Forbidden", nil)
		return
	}
	password := s.password
	if device != nil && device.Password != "" {
		password = device.Password
	}
	if password == "" && device == nil {
		s.respond(tx, req, 403, "Forbidden", nil)
		return
	}

	auth := req.GetHeader("Authorization")
	if auth == nil {
		ch := NewDigestChallenge(s.cfg.Domain)
		s.challenges.Store(deviceID, ch)
		res := sip.NewResponseFromRequest(req, 401, "Unauthorized", nil)
		res.AppendHeader(sip.NewHeader("WWW-Authenticate", ch.String()))
		_ = tx.Respond(res)
		return
	}

	chVal, _ := s.challenges.Load(deviceID)
	ch, _ := chVal.(DigestChallenge)
	authParams := ParseAuthorization(auth.Value())
	uri := authParams["uri"]
	if uri == "" {
		uri = req.Recipient.String()
	}
	if !VerifyDigest(auth.Value(), "REGISTER", uri, deviceID, password, ch.Realm, ch.Nonce) {
		log.Printf("GB28181 register auth failed: id=%s ip=%s:%d uri=%s qop=%s", deviceID, ip, port, uri, authParams["qop"])
		s.respond(tx, req, 403, "Forbidden", nil)
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	if device == nil {
		device = &domaindevice.Device{
			DeviceID:          deviceID,
			Name:              deviceID,
			Transport:         strings.ToUpper(string(req.Transport())),
			StreamMode:        "UDP", // RTP 媒体默认 UDP，与 SIP 信令走 UDP/TCP 无关
			Charset:           "GB2312",
			MediaServerID:     "auto",
			HeartBeatInterval: 60,
			HeartBeatCount:    3,
			ServerID:          s.serverID,
			CreateTime:        now,
		}
	}
	device.IP = ip
	device.Port = port
	device.HostAddress = ip + ":" + strconv.Itoa(port)
	device.Expires = expires
	device.Transport = strings.ToUpper(string(req.Transport()))
	// 仅更新信令传输方式，不覆盖 streamMode（由平台「流传输模式」或前端单独配置）
	device.UpdateTime = now
	if callID := req.CallID(); callID != nil {
		device.RegisterCallID = callID.Value()
	}
	// 记录平台侧可达 IP，供 INVITE Contact 使用
	if lip := s.resolveInviteLocalIP(device); lip != "" {
		device.LocalIP = lip
	}

	saved, err := s.deviceSvc.SaveRegister(device)
	if err != nil {
		log.Printf("save register device %s: %v", deviceID, err)
		s.respond(tx, req, 500, "Server Error", nil)
		return
	}
	_ = s.deviceSvc.Online(saved)
	s.deviceSvc.OnDeviceOnline(saved)
	if s.redis != nil {
		_ = s.redis.PushRegister(context.Background(), deviceID, time.Now().UnixMilli())
	}

	res := sip.NewResponseFromRequest(req, 200, "OK", nil)
	res.AppendHeader(sip.NewHeader("Date", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")))
	if exp := sip.NewHeader("Expires", strconv.Itoa(expires)); exp != nil {
		res.AppendHeader(exp)
	}
	_ = tx.Respond(res)
	log.Printf("GB28181 device registered: id=%s ip=%s:%d transport=%s streamMode=%s expires=%d",
		deviceID, ip, port, device.Transport, device.StreamMode, expires)
}

func (s *Server) handleUnregister(req *sip.Request, tx sip.ServerTransaction, deviceID string) {
	if device, err := s.deviceSvc.GetByDeviceID(deviceID); err == nil {
		_ = s.deviceSvc.Offline(device)
	}
	s.respond(tx, req, 200, "OK", nil)
}

func (s *Server) handleMessage(req *sip.Request, tx sip.ServerTransaction) {
	fromDeviceID := extractSIPUser(req)
	body := req.Body()
	msg, err := ParseGBXML(body)
	if err != nil {
		s.respond(tx, req, 400, "Bad Request", nil)
		return
	}
	deviceID := fromDeviceID
	if deviceID == "" {
		deviceID = msg.DeviceID
	}

	device, err := s.deviceSvc.GetByDeviceID(deviceID)
	if err != nil || device == nil {
		s.respond(tx, req, 404, "Not Found", nil)
		return
	}

	switch msg.Root {
	case "Notify":
		switch msg.CmdType {
		case "Keepalive":
			ip, port := extractSourceAddr(req)
			_ = s.deviceSvc.HandleKeepalive(deviceID, ip, port)
			if s.redis != nil {
				_ = s.redis.PushKeepalive(context.Background(), deviceID, time.Now().UnixMilli())
			}
		case "Catalog":
			log.Printf("GB28181 catalog notify: device=%s items=%d", deviceID, len(msg.Items))
			_ = s.deviceSvc.HandleCatalog(deviceID, msg.Items)
		case "Alarm":
			if s.alarmHandler != nil {
				_ = s.alarmHandler.HandleNotify(deviceID, msg.DeviceID, msg.Alarm)
			}
		case "MobilePosition":
			if s.positionHandler != nil {
				_ = s.positionHandler.HandleNotify(deviceID, msg.DeviceID, msg.Position)
			}
		}
	case "Response":
		switch msg.CmdType {
		case "Catalog":
			log.Printf("GB28181 catalog response: device=%s items=%d", deviceID, len(msg.Items))
			_ = s.deviceSvc.HandleCatalog(deviceID, msg.Items)
		case "DeviceInfo":
			devName := extractTag(body, "DeviceName")
			if devName == "" {
				devName = extractTag(body, "Name")
			}
			mfr := extractTag(body, "Manufacturer")
			model := extractTag(body, "Model")
			fw := extractTag(body, "Firmware")
			log.Printf("GB28181 DeviceInfo response: device=%s name=%s manufacturer=%s model=%s",
				deviceID, devName, mfr, model)
			_ = s.deviceSvc.HandleDeviceInfo(deviceID, devName, mfr, model, fw)
		case "RecordInfo":
			s.recordMgr.HandleRecordInfo(deviceID, msg.DeviceID, msg.SN, msg.SumNum, msg.RecordItems)
		case "PresetQuery":
			s.presetMgr.HandlePresetQuery(msg.SN, msg.SumNum, msg.PresetItems)
		}
	}

	s.respond(tx, req, 200, "OK", nil)
}

func (s *Server) SendCatalogQuery(device *domaindevice.Device) error {
	sn := strconv.FormatInt(s.sn.Add(1), 10)
	body := BuildCatalogQuery(device.DeviceID, s.cfg.ID, sn)
	return s.sendMessage(device, body)
}

func (s *Server) SendDeviceInfoQuery(device *domaindevice.Device) error {
	sn := strconv.FormatInt(s.sn.Add(1), 10)
	body := BuildDeviceInfoQuery(device.DeviceID, sn)
	return s.sendMessage(device, body)
}

func (s *Server) SendDeviceControl(device *domaindevice.Device, channelID, xmlBody string) error {
	return s.sendMessage(device, xmlBody)
}

func (s *Server) SendInvitePlay(device *domaindevice.Device, channel *domainchannel.Channel, sdp, ssrc, stream, streamMode string, tcpConnect func(host string, port int) error, onOK func(*sip.Response)) error {
	sess, err := s.inviteDialog(device, channel, sdp, ssrc)
	if err != nil {
		return err
	}
	if strings.EqualFold(streamMode, "TCP-ACTIVE") && tcpConnect != nil && sess.InviteResponse != nil {
		host, port, parseErr := ParseInviteAnswerMedia(string(sess.InviteResponse.Body()))
		if parseErr != nil {
			log.Printf("[GB28181 sip] TCP-ACTIVE parse 200 OK SDP failed: %v", parseErr)
			sess.Close()
			return parseErr
		}
		log.Printf("[GB28181 sip] TCP-ACTIVE connectRtpServer -> %s:%d", host, port)
		if err := tcpConnect(host, port); err != nil {
			log.Printf("[GB28181 sip] TCP-ACTIVE connectRtpServer failed: %v", err)
			sess.Close()
			return err
		}
	}
	s.inviteMgr.Put(stream, &InviteSession{
		Device: device, Channel: channel, Stream: stream, App: "live", Type: SessionPlay,
		Dialog: sess, StartedAt: time.Now(),
	})
	log.Printf("[GB28181 sip] invite session stored stream=%s (keep alive until stop/BYE)", stream)
	if onOK != nil && sess.InviteResponse != nil {
		onOK(sess.InviteResponse)
	}
	return nil
}

func (s *Server) SendInviteSession(device *domaindevice.Device, channel *domainchannel.Channel, sdp, ssrc, stream string, sessionType SessionType, startTime, endTime string, downloadSpeed int) error {
	sess, err := s.inviteDialog(device, channel, sdp, ssrc)
	if err != nil {
		return err
	}
	s.inviteMgr.Put(stream, &InviteSession{
		Device: device, Channel: channel, Stream: stream, App: "live", Type: sessionType,
		Dialog: sess, StartTime: startTime, EndTime: endTime, DownloadSpeed: downloadSpeed,
		StartedAt: time.Now(),
	})
	return nil
}

func (s *Server) SendPlaybackControl(stream, content string) error {
	sess, ok := s.inviteMgr.Get(stream)
	if !ok {
		return ErrSessionNotFound
	}
	recipient := sess.Dialog.InviteRequest.Recipient
	req := sip.NewRequest(sip.INFO, recipient)
	req.SetBody([]byte(content))
	req.AppendHeader(sip.NewHeader("Content-Type", "Application/MANSRTSP"))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := sess.Dialog.Do(ctx, req)
	return err
}

func (s *Server) CloseInviteSession(stream string) error {
	sess, ok := s.inviteMgr.Get(stream)
	if !ok {
		return nil
	}
	defer s.inviteMgr.Remove(stream)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sess.Dialog.Bye(ctx); err != nil {
		_ = sess.Dialog.Close()
	}
	return nil
}

func (s *Server) InviteManager() *InviteManager { return s.inviteMgr }

func (s *Server) NextInfoCSeq() int {
	return int(s.infoCSeq.Add(1))
}

func (s *Server) resolveInviteLocalIP(device *domaindevice.Device) string {
	candidates := []string{s.localIP}
	if device != nil {
		candidates = append(candidates, device.LocalIP, device.SDPIP)
	}
	for _, ip := range candidates {
		ip = strings.TrimSpace(ip)
		if ip != "" && ip != "0.0.0.0" && ip != "127.0.0.1" {
			return ip
		}
	}
	if device != nil && device.IP != "" {
		if ip := detectLocalIPForRemote(device.IP); ip != "" {
			return ip
		}
	}
	return guessLocalIP()
}

func (s *Server) inviteDialog(device *domaindevice.Device, channel *domainchannel.Channel, sdp, ssrc string) (*sipgo.DialogClientSession, error) {
	localIP := s.resolveInviteLocalIP(device)
	if localIP == "" {
		return nil, fmt.Errorf("sip.ip 未配置：INVITE Contact 需要平台可达 IP，请在 config.yaml 设置 sip.ip")
	}
	dialogUA := &sipgo.DialogUA{
		Client: s.client,
		ContactHDR: sip.ContactHeader{
			Address: sip.Uri{User: s.cfg.ID, Host: localIP, Port: s.cfg.Port},
		},
	}
	channelID := channel.GBDeviceID
	recipient := sip.Uri{User: channelID, Host: device.IP, Port: device.Port}
	subject := fmt.Sprintf("%s:%s,%s:0", channelID, ssrc, s.cfg.ID)
	log.Printf("[GB28181 sip] INVITE %s@%s:%d Subject=%s localIP=%s sdpPort=%d",
		channelID, device.IP, device.Port, subject, localIP, extractSDPPort(sdp))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sess, err := dialogUA.Invite(ctx, recipient, []byte(sdp),
		sip.NewHeader("Content-Type", "APPLICATION/SDP"),
		sip.NewHeader("Subject", subject),
	)
	if err != nil {
		log.Printf("[GB28181 sip] INVITE send error: %v", err)
		return nil, err
	}
	if err := sess.WaitAnswer(ctx, sipgo.AnswerOptions{}); err != nil {
		log.Printf("[GB28181 sip] INVITE no 200 OK: %v", err)
		sess.Close()
		return nil, err
	}
	log.Printf("[GB28181 sip] INVITE 200 OK status=%d", sess.InviteResponse.StatusCode)
	if err := sess.Ack(ctx); err != nil {
		log.Printf("[GB28181 sip] ACK error: %v", err)
		sess.Close()
		return nil, err
	}
	log.Printf("[GB28181 sip] ACK sent, waiting camera RTP/PS (streamMode in offer SDP)")
	return sess, nil
}

// ParseInviteAnswerMedia 从 INVITE 200 OK 的 SDP 解析摄像机媒体地址（TCP-ACTIVE 用）。
func ParseInviteAnswerMedia(sdp string) (host string, port int, err error) {
	sdp = strings.ReplaceAll(sdp, "\r\n", "\n")
	for _, line := range strings.Split(sdp, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "c=IN IP4 ") {
			host = strings.TrimSpace(strings.TrimPrefix(line, "c=IN IP4 "))
		}
		if strings.HasPrefix(line, "m=video ") {
			var mediaPort int
			if _, scanErr := fmt.Sscanf(line, "m=video %d", &mediaPort); scanErr == nil && mediaPort > 0 {
				port = mediaPort
			}
		}
	}
	if host == "" || port <= 0 {
		return "", 0, fmt.Errorf("answer SDP missing c= or m=video port")
	}
	return host, port, nil
}

func extractSDPPort(sdp string) int {
	for _, line := range strings.Split(sdp, "\n") {
		if strings.HasPrefix(line, "m=video ") {
			var port int
			if _, err := fmt.Sscanf(line, "m=video %d", &port); err == nil {
				return port
			}
		}
	}
	return 0
}

func (s *Server) SendPTZ(device *domaindevice.Device, channelID, direction string, h, v, z int) error {
	cmd := BuildDeviceControlPTZ(device.DeviceID, channelID, PTZCommand(direction, h, v, z))
	return s.sendMessage(device, cmd)
}

func (s *Server) SendAudioBroadcast(device *domaindevice.Device, channelGBID string) error {
	sn := strconv.FormatInt(s.sn.Add(1), 10)
	body := BuildBroadcastNotify(s.cfg.ID, channelGBID, sn)
	return s.sendMessage(device, body)
}

func (s *Server) sendMessage(device *domaindevice.Device, body string) error {
	recipient := sip.Uri{User: device.DeviceID, Host: device.IP, Port: device.Port}
	req := sip.NewRequest(sip.MESSAGE, recipient)
	req.SetBody([]byte(body))
	req.AppendHeader(sip.NewHeader("Content-Type", "Application/MANSCDP+xml"))
	from := &sip.FromHeader{Address: sip.Uri{User: s.cfg.ID, Host: s.cfg.Domain}}
	to := &sip.FromHeader{Address: recipient}
	req.AppendHeader(from)
	req.AppendHeader(to)
	return s.client.WriteRequest(req)
}

func (s *Server) respond(tx sip.ServerTransaction, req *sip.Request, code int, reason string, _ []byte) {
	res := sip.NewResponseFromRequest(req, code, reason, nil)
	_ = tx.Respond(res)
}

func extractSIPUser(req *sip.Request) string {
	if from := req.From(); from != nil {
		return from.Address.User
	}
	return ""
}

func parseExpires(req *sip.Request) int {
	if h := req.GetHeader("Expires"); h != nil {
		if v, err := strconv.Atoi(strings.TrimSpace(h.Value())); err == nil {
			return v
		}
	}
	if c := req.Contact(); c != nil && c.Address.UriParams != nil {
		if exp, ok := c.Address.UriParams.Get("expires"); ok && exp != "" {
			if v, err := strconv.Atoi(exp); err == nil {
				return v
			}
		}
	}
	return 3600
}

func extractSourceAddr(req *sip.Request) (string, int) {
	host, portStr, err := net.SplitHostPort(req.Source())
	if err != nil {
		return req.Source(), 5060
	}
	port, _ := strconv.Atoi(portStr)
	return host, port
}
