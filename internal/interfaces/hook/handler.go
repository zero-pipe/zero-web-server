package hook

import (
	cloudrecordapp "zero-web-server/internal/application/cloudrecord"
	mediaserverapp "zero-web-server/internal/application/mediaserver"
	playapp "zero-web-server/internal/application/play"
	playbackapp "zero-web-server/internal/application/playback"
	publishauth "zero-web-server/internal/application/publishauth"
	recordplanapp "zero-web-server/internal/application/recordplan"
	streamproxyapp "zero-web-server/internal/application/streamproxy"
	streampushapp "zero-web-server/internal/application/streampush"
	applog "zero-web-server/pkg/log"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type StreamNotifier interface {
	OnStreamStarted(app, stream string)
}

type Handler struct {
	notifiers      []StreamNotifier
	play           *playapp.Service
	cloudRecord    *cloudrecordapp.Service
	streamPush     *streampushapp.Service
	streamProxy    *streamproxyapp.Service
	recordPlan     *recordplanapp.Service
	publishAuth    *publishauth.PublishAuth
	mediaServer    *mediaserverapp.Service
	streamOnDemand bool
	// accessTokenCheck(token) → (有效, 是否已启用校验)；nil 表示不做接入 token 校验。
	accessTokenCheck func(token string) (bool, bool)
}

func NewHandler(
	play *playapp.Service,
	playback *playbackapp.Service,
	cloud *cloudrecordapp.Service,
	push *streampushapp.Service,
	proxy *streamproxyapp.Service,
	plan *recordplanapp.Service,
	mediaServer *mediaserverapp.Service,
	publishAuth *publishauth.PublishAuth,
	streamOnDemand bool,
	accessTokenCheck func(token string) (bool, bool),
) *Handler {
	notifiers := make([]StreamNotifier, 0, 4)
	for _, n := range []StreamNotifier{play, playback, cloud, push, proxy} {
		if n != nil {
			notifiers = append(notifiers, n)
		}
	}
	return &Handler{
		notifiers: notifiers, play: play, cloudRecord: cloud,
		streamPush: push, streamProxy: proxy, recordPlan: plan,
		mediaServer:       mediaServer,
		publishAuth:       publishAuth,
		streamOnDemand:    streamOnDemand,
		accessTokenCheck:  accessTokenCheck,
	}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	r.POST("/on_publish", h.onPublish)
	r.POST("/on_play", h.onPlay)
	r.POST("/on_stream_changed", h.onStreamChanged)
	r.POST("/on_stream_not_found", h.onStreamNotFound)
	r.POST("/on_stream_none_reader", h.onStreamNoneReader)
	r.POST("/on_record_mp4", h.onRecordMp4)
	r.POST("/on_server_started", h.onServerStarted)
	r.POST("/on_server_keepalive", h.onServerKeepalive)
}

// onOK 应答媒体服务器期望的标准成功包。
func (h *Handler) onOK(c *gin.Context) {
	c.JSON(200, gin.H{"code": 0, "msg": "success"})
}

// onServerStarted 媒体服务器启动自注册：解析上报的端口/能力并 upsert 节点，
// 免去页面手工添加媒体节点。payload 见 RegisterRequest。
func (h *Handler) onServerStarted(c *gin.Context) {
	if h.mediaServer == nil {
		h.onOK(c)
		return
	}
	var req mediaserverapp.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		applog.Warnf("[media] on_server_started 解析失败 remote=%s err=%v", c.ClientIP(), err)
		h.onOK(c)
		return
	}
	id, err := h.mediaServer.Register(c.Request.Context(), req)
	if err != nil {
		applog.Warnf("[media] on_server_started 注册失败 remote=%s err=%v", c.ClientIP(), err)
		h.onOK(c)
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "success", "mediaServerId": id})
}

// onServerKeepalive 媒体服务器心跳：刷新在线与 LastSeen；并可携带完整注册信息以便续注册。
func (h *Handler) onServerKeepalive(c *gin.Context) {
	if h.mediaServer == nil {
		h.onOK(c)
		return
	}
	var req mediaserverapp.RegisterRequest
	if err := c.ShouldBindJSON(&req); err == nil && req.HTTPPort > 0 && (req.IP != "" || req.StreamIP != "") {
		if id, err := h.mediaServer.Register(c.Request.Context(), req); err == nil {
			c.JSON(200, gin.H{"code": 0, "msg": "success", "mediaServerId": id})
			return
		}
	}
	// 精简心跳：仅刷新 id（ZMS 正常会带完整注册体走上面的 Register 分支）。
	_ = h.mediaServer.Touch(req.MediaServerID)
	h.onOK(c)
}

func (h *Handler) onPublish(c *gin.Context) {
	fields := readHookFields(c)
	app := fields.App
	if app == "" {
		app = c.PostForm("app")
	}
	stream := fields.Stream
	if stream == "" {
		stream = c.PostForm("stream")
	}
	mediaServerID := c.PostForm("mediaServerId")
	if mediaServerID == "" {
		mediaServerID = c.Query("mediaServerId")
	}
	params := c.PostForm("params")
	if token := extractHookToken(c, params); token != "" && h.accessTokenCheck != nil {
		// 接入 token 已在平台登记并启用时，推流必须携带有效 token。
		valid, enforcing := h.accessTokenCheck(token)
		if enforcing && !valid {
			c.JSON(200, gin.H{"code": -1, "msg": "invalid access token"})
			return
		}
	}
	if h.streamPush != nil {
		h.streamPush.OnPublish(app, stream, mediaServerID)
	}
	if h.publishAuth == nil {
		c.JSON(200, gin.H{"code": 0, "msg": "success", "enable_audio": true, "enable_mp4": false})
		return
	}
	result := h.publishAuth.Authenticate(app, stream)
	if !result.Allowed {
		c.JSON(200, gin.H{"code": -1, "msg": "Unauthorized"})
		return
	}
	c.JSON(200, gin.H{
		"code": 0, "msg": "success",
		"enable_audio": result.EnableAudio,
		"enable_mp4":   result.EnableMP4,
	})
}

func (h *Handler) onPlay(c *gin.Context) {
	h.onOK(c)
}

func (h *Handler) onStreamChanged(c *gin.Context) {
	fields := readHookFields(c)
	app, stream := fields.App, fields.Stream
	applog.Debugf("[GB28181 hook] on_stream_changed app=%s stream=%s regist=%v remote=%s",
		app, stream, fields.Regist, c.ClientIP())
	if fields.Regist {
		for _, n := range h.notifiers {
			n.OnStreamStarted(app, stream)
		}
		if h.play != nil {
			h.play.OnBroadcastStreamArrival(app, stream)
		}
	} else {
		if h.streamPush != nil {
			h.streamPush.OnStreamDeparture(app, stream)
		}
		if h.recordPlan != nil {
			h.recordPlan.OnStreamDeparture(app, stream)
		}
	}
	response.OK(c, gin.H{"code": 0, "msg": "success"})
}

func (h *Handler) onStreamNotFound(c *gin.Context) {
	h.onOK(c)
}

func (h *Handler) onStreamNoneReader(c *gin.Context) {
	fields := readHookFields(c)
	app, stream := fields.App, fields.Stream
	if app == "" {
		app = c.Query("app")
	}
	if stream == "" {
		stream = c.Query("stream")
	}
	closeStream := h.streamOnDemand
	if publishauth.IsGBLiveApp(app) {
		// GB28181 由 Go SIP 会话管理；切换播放器时会出现短暂无人观看，勿关流
		closeStream = false
	} else if app == "onvif" {
		// ONVIF RTSP 拉流代理：切换 Jessibuca/H265web 时会有短暂无人观看，勿关流
		closeStream = false
	} else if h.recordPlan != nil && h.recordPlan.Recording(app, stream) {
		closeStream = false
	} else if app == publishauth.LoadMP4App {
		closeStream = false
	} else if h.streamProxy != nil {
		if v := h.streamProxy.CloseOnNoneReader(app, stream); v != nil {
			closeStream = *v
		}
	}
	applog.Debugf("[GB28181 hook] on_stream_none_reader app=%s stream=%s close=%v (stream_on_demand=%v)",
		app, stream, closeStream, h.streamOnDemand)
	c.JSON(200, gin.H{"code": 0, "close": closeStream})
}

func (h *Handler) onRecordMp4(c *gin.Context) {
	if h.cloudRecord == nil {
		h.onOK(c)
		return
	}
	fields := readRecordMp4Fields(c)
	param := cloudrecordapp.RecordHookParam{
		App: fields.App, Stream: fields.Stream,
		FileName: fields.FileName, FilePath: fields.FilePath, URL: fields.URL,
		FileSize: fields.FileSize, Folder: fields.Folder,
		StartTime: fields.StartTime, TimeLen: fields.TimeLen,
		CallID: fields.CallID, MediaServerID: fields.MediaServerID,
	}
	_ = h.cloudRecord.OnRecordMp4(param)
	h.onOK(c)
}
