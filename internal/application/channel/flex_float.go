package channelapp

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

// FlexFloat 兼容 JSON 数字与字符串（前端 el-input 常把经纬度等变成字符串）。
type FlexFloat float64

func (f *FlexFloat) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		*f = 0
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		if s == "" {
			*f = 0
			return nil
		}
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		*f = FlexFloat(v)
		return nil
	}
	var v float64
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*f = FlexFloat(v)
	return nil
}

func (f FlexFloat) MarshalJSON() ([]byte, error) {
	return json.Marshal(float64(f))
}

func (f FlexFloat) Float64() float64 { return float64(f) }
