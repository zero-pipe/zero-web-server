package hook

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
)

type hookFields struct {
	App    string
	Stream string
	Regist bool
}

func readHookFields(c *gin.Context) hookFields {
	if strings.Contains(c.GetHeader("Content-Type"), "application/json") {
		body, err := io.ReadAll(c.Request.Body)
		if err == nil && len(body) > 0 {
			var payload map[string]json.RawMessage
			if json.Unmarshal(body, &payload) == nil {
				return hookFields{
					App:    hookString(payload["app"]),
					Stream: hookString(payload["stream"]),
					Regist: hookRegist(payload["regist"]),
				}
			}
		}
	}
	return hookFields{
		App:    c.PostForm("app"),
		Stream: c.PostForm("stream"),
		Regist: hookRegistString(c.PostForm("regist")),
	}
}

func hookString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	return ""
}

func hookRegist(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var b bool
	if json.Unmarshal(raw, &b) == nil {
		return b
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return hookRegistString(s)
	}
	var n float64
	if json.Unmarshal(raw, &n) == nil {
		return n == 1
	}
	return false
}

func hookRegistString(v string) bool {
	return v == "1" || strings.EqualFold(v, "true")
}
