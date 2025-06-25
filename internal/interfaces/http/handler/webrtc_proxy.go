package handler



import (

	"bytes"

	"io"

	"log"

	"net/http"

	"strings"

	"time"



	"github.com/gin-gonic/gin"

)



// WebRTCProxyHandler 将 WebRTC 信令转发到 ZMS/ZLM，供前端 rtc 字段走平台同源地址。

type WebRTCProxyHandler struct {

	mediaBaseURL string

	client       *http.Client

}



func NewWebRTCProxyHandler(mediaBaseURL string) *WebRTCProxyHandler {

	return &WebRTCProxyHandler{

		mediaBaseURL: strings.TrimRight(mediaBaseURL, "/"),

		client:       &http.Client{Timeout: 30 * time.Second},

	}

}



func (h *WebRTCProxyHandler) Proxy(c *gin.Context) {

	target := h.mediaBaseURL + c.Request.URL.Path

	if raw := c.Request.URL.RawQuery; raw != "" {

		target += "?" + raw

	}



	body, err := io.ReadAll(c.Request.Body)

	if err != nil {

		c.JSON(http.StatusBadGateway, gin.H{"code": -1, "msg": err.Error()})

		return

	}



	req, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, target, bytes.NewReader(body))

	if err != nil {

		c.JSON(http.StatusBadGateway, gin.H{"code": -1, "msg": err.Error()})

		return

	}

	req.ContentLength = int64(len(body))

	if ct := c.GetHeader("Content-Type"); ct != "" {

		req.Header.Set("Content-Type", ct)

	} else if len(body) > 0 {

		req.Header.Set("Content-Type", "text/plain;charset=utf-8")

	}



	resp, err := h.client.Do(req)

	if err != nil {

		log.Printf("[webrtc-proxy] %s %s body=%d -> %v", c.Request.Method, target, len(body), err)

		c.JSON(http.StatusBadGateway, gin.H{"code": -1, "msg": err.Error()})

		return

	}

	defer resp.Body.Close()



	log.Printf("[webrtc-proxy] %s %s body=%d -> %d", c.Request.Method, target, len(body), resp.StatusCode)



	for k, vals := range resp.Header {

		for _, v := range vals {

			c.Header(k, v)

		}

	}

	c.Status(resp.StatusCode)

	if _, err := io.Copy(c.Writer, resp.Body); err != nil {

		log.Printf("[webrtc-proxy] copy response: %v", err)

	}

}

