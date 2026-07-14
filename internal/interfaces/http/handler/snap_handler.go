package handler

import (
	"io"
	"net/http"

	snapapp "zero-web-kit/internal/application/snap"
	"zero-web-kit/pkg/response"

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
		response.Error(c, response.CodeError, err.Error())
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	key, url, err := h.snap.PutJPEG(c.Request.Context(), deviceID, channelID, data)
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	response.OK(c, gin.H{"key": key, "url": url})
}
