package handler

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"zero-web-kit/internal/application/ops"
	"zero-web-kit/pkg/response"

	"github.com/gin-gonic/gin"
)

type LogHandler struct {
	svc *ops.LogService
}

func NewLogHandler(svc *ops.LogService) *LogHandler {
	return &LogHandler{svc: svc}
}

func (h *LogHandler) List(c *gin.Context) {
	list, err := h.svc.List(c.Query("query"), c.Query("startTime"), c.Query("endTime"))
	if err != nil {
		response.Error(c, response.CodeError, err.Error())
		return
	}
	if list == nil {
		list = []ops.LogFileInfo{}
	}
	response.OK(c, list)
}

func (h *LogHandler) File(c *gin.Context) {
	name := c.Param("fileName")
	name = strings.TrimPrefix(name, "/")
	path, err := h.svc.ResolveFile(name)
	if err != nil {
		response.Error(c, response.CodeBadReq, err.Error())
		return
	}
	f, err := os.Open(path)
	if err != nil {
		response.Error(c, response.CodeError, "打开文件失败")
		return
	}
	defer f.Close()

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Content-Disposition", "inline; filename=\""+filepath.Base(path)+"\"")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, f)
}
