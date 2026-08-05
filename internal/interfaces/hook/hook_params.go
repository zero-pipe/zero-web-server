package hook

import (
	"encoding/json"
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type hookFields struct {
	App    string
	Stream string
	Regist bool
}

type recordMp4Fields struct {
	App           string
	Stream        string
	FileName      string
	FilePath      string
	URL           string
	Folder        string
	CallID        string
	MediaServerID string
	FileSize      int64
	StartTime     int64
	TimeLen       float64
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

func readRecordMp4Fields(c *gin.Context) recordMp4Fields {
	out := recordMp4Fields{}
	if strings.Contains(c.GetHeader("Content-Type"), "application/json") {
		body, err := io.ReadAll(c.Request.Body)
		if err == nil && len(body) > 0 {
			var payload map[string]json.RawMessage
			if json.Unmarshal(body, &payload) == nil {
				out.App = hookString(payload["app"])
				out.Stream = hookString(payload["stream"])
				out.FileName = firstNonEmpty(hookString(payload["file_name"]), hookString(payload["fileName"]))
				out.FilePath = firstNonEmpty(hookString(payload["file_path"]), hookString(payload["filePath"]))
				out.URL = firstNonEmpty(hookString(payload["url"]), hookString(payload["play_url"]), hookString(payload["playUrl"]))
				out.Folder = hookString(payload["folder"])
				out.CallID = firstNonEmpty(hookString(payload["call_id"]), hookString(payload["callId"]))
				out.MediaServerID = firstNonEmpty(
					hookString(payload["mediaServerId"]),
					hookString(payload["media_server_id"]),
				)
				out.FileSize = hookInt64(payload["file_size"], payload["fileSize"])
				out.StartTime = hookInt64(payload["start_time"], payload["startTime"])
				out.TimeLen = hookFloat64(payload["time_len"], payload["timeLen"])
				return out
			}
		}
	}
	out.App = c.PostForm("app")
	out.Stream = c.PostForm("stream")
	out.FileName = firstNonEmpty(c.PostForm("file_name"), c.PostForm("fileName"))
	out.FilePath = firstNonEmpty(c.PostForm("file_path"), c.PostForm("filePath"))
	out.URL = firstNonEmpty(c.PostForm("url"), c.PostForm("play_url"), c.PostForm("playUrl"), c.Query("url"))
	out.Folder = c.PostForm("folder")
	out.CallID = firstNonEmpty(c.PostForm("call_id"), c.PostForm("callId"))
	out.MediaServerID = firstNonEmpty(c.PostForm("mediaServerId"), c.Query("mediaServerId"))
	out.FileSize, _ = strconv.ParseInt(firstNonEmpty(c.PostForm("file_size"), c.PostForm("fileSize")), 10, 64)
	out.StartTime, _ = strconv.ParseInt(firstNonEmpty(c.PostForm("start_time"), c.PostForm("startTime")), 10, 64)
	out.TimeLen, _ = strconv.ParseFloat(firstNonEmpty(c.PostForm("time_len"), c.PostForm("timeLen")), 64)
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
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

func hookInt64(raws ...json.RawMessage) int64 {
	for _, raw := range raws {
		if len(raw) == 0 {
			continue
		}
		var n float64
		if json.Unmarshal(raw, &n) == nil {
			return int64(n)
		}
		var s string
		if json.Unmarshal(raw, &s) == nil {
			v, err := strconv.ParseInt(s, 10, 64)
			if err == nil {
				return v
			}
		}
	}
	return 0
}

func hookFloat64(raws ...json.RawMessage) float64 {
	for _, raw := range raws {
		if len(raw) == 0 {
			continue
		}
		var n float64
		if json.Unmarshal(raw, &n) == nil {
			return n
		}
		var s string
		if json.Unmarshal(raw, &s) == nil {
			v, err := strconv.ParseFloat(s, 64)
			if err == nil {
				return v
			}
		}
	}
	return 0
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
