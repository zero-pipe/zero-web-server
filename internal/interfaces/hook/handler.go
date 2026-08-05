package hook

import (
	cloudrecordapp "zero-web-server/internal/application/cloudrecord"
	publishauth "zero-web-server/internal/application/publishauth"
	playapp "zero-web-server/internal/application/play"
	playbackapp "zero-web-server/internal/application/playback"
	recordplanapp "zero-web-server/internal/application/recordplan"
	streampushapp "zero-web-server/internal/application/streampush"
	streamproxyapp "zero-web-server/internal/application/streamproxy"
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
	streamOnDemand bool
}

func NewHandler(
	play *playapp.Service,
	playback *playbackapp.Service,
	cloud *cloudrecordapp.Service,
	push *streampushapp.Service,
	proxy *streamproxyapp.Service,
	plan *recordplanapp.Service,
	publishAuth *publishauth.PublishAuth,
	streamOnDemand bool,
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
		publishAuth: publishAuth, streamOnDemand: streamOnDemand,
	}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	r.POST("/on_publish", h.onPublish)
	r.POST("/on_play", h.onPlay)
	r.POST("/on_stream_changed", h.onStreamChanged)
	r.POST("/on_stream_not_found", h.onStreamNotFound)
	r.POST("/on_stream_none_reader", h.onStreamNoneReader)
	r.POST("/on_record_mp4", h.onRecordMp4)
	r.POST("/on_server_started", h.onOK)
	r.POST("/on_server_keepalive", h.onOK)
}

func (h *Handler) onOK(c *gin.Context) {
	c.JSON(200, gin.H{"code": 0, "msg": "success"})
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
	if h.streamPush != nil {
		h.streamPush.OnPublish(app, stream, mediaServerID)
	}
	if h.publishAuth == nil {
		c.JSON(200, gin.H{"code": 0, "msg": "success", "enable_audio": true, "enable_mp4": false})
		return
	}
	result := h.publishAuth.Authenticate(app, stream, params)
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
