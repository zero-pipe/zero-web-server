package handler

import (
	"io"
	"net/http"
	"strings"

	snapapp "zero-web-server/internal/application/snap"
	"zero-web-server/internal/gbsip"
	"zero-web-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type SnapHandler struct {
	snap *snapapp.Service
}

func NewSnapHandler(snap *snapapp.Service) *SnapHandler {
	return &SnapHandler{snap: snap}
}

// GetChannelSnap GET /api/device/query/snap/:deviceId/:channelId
func (h *SnapHandler) GetChannelSnap(c *gin.Context) {
	deviceID := c.Param("deviceId")
	channelID := c.Param("channelId")
	body, ct, redirect, err := h.snap.OpenLatest(c.Request.Context(), deviceID, channelID)
	if err != nil {
		c.Status(http.StatusNoContent)
		return
	}
	if redirect != "" {
		c.Redirect(http.StatusFound, redirect)
		return
	}
	defer body.Close()
	if ct == "" {
		ct = "image/jpeg"
	}
	c.Header("Content-Type", ct)
	_, _ = io.Copy(c.Writer, body)
}

// UploadChannelSnap POST multipart field "file"
func (h *SnapHandler) UploadChannelSnap(c *gin.Context) {
	deviceID := c.Param("deviceId")
	channelID := c.Param("channelId")
	fh, err := c.FormFile("file")
	if err != nil {
		response.Error(c, response.CodeBadReq, "缺少 file")
		return
	}
	f, err := fh.Open()
	if err != nil {
		response.FromError(c, err)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		response.FromError(c, err)
		return
	}
	key, url, err := h.snap.PutJPEG(c.Request.Context(), deviceID, channelID, data)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"key": key, "url": url})
}

// UploadGBSnapshot POST /api/v1/gb/snapshot/upload?deviceId=&sessionId=&fileName=
// 公开接口：设备按图像抓拍配置的 UploadURL 上传 JPEG（2022 9.14）。
func (h *SnapHandler) UploadGBSnapshot(c *gin.Context) {
	data, err := io.ReadAll(io.LimitReader(c.Request.Body, 20<<20))
	if err != nil || len(data) == 0 {
		response.Error(c, response.CodeBadReq, "空图像")
		return
	}
	fileName := strings.TrimSpace(c.Query("fileName"))
	deviceID := strings.TrimSpace(c.Query("deviceId"))
	if deviceID == "" && fileName != "" {
		if id, _, _, ok := gbsip.ParseSnapShotFileName(fileName); ok {
			deviceID = id
		}
	}
	if deviceID == "" {
		response.Error(c, response.CodeBadReq, "缺少 deviceId 或可解析的 fileName")
		return
	}
	key, url, err := h.snap.PutSnapshot(c.Request.Context(), deviceID, fileName, data)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"key": key, "url": url, "sessionId": c.Query("sessionId")})
}

// Finished GET /api/device/query/snap/finished/:sessionId
func (h *SnapHandler) Finished(c *gin.Context) {
	rec, ok := h.snap.Finished(c.Param("sessionId"))
	if !ok {
		response.Error(c, response.CodeError, "暂无完成通知")
		return
	}
	response.OK(c, rec)
}
