package dto

import domainchannel "zero-web-server/internal/domain/channel"

// 国标目录 Info/PTZType 枚举（2016 A.2.1 g、2022 A.2.1.9）。
var ptzTypeTexts = map[int]string{
	0: "未知",
	1: "球机",
	2: "半球",
	3: "固定枪机",
	4: "遥控枪机",
	5: "遥控半球",
	6: "全景通道",
	7: "分割通道",
}

type ChannelView struct {
	ID                   int     `json:"id"`
	InternalCode         string  `json:"internalCode"`
	ChannelID            int     `json:"channelId"`
	DeviceID             string  `json:"deviceId"`
	ParentDeviceID       string  `json:"parentDeviceId"`
	Name                 string  `json:"name"`
	Manufacturer         string  `json:"manufacturer"`
	Model                string  `json:"model"`
	Status               string  `json:"status"`
	PTZType              int     `json:"ptzType"`
	PTZTypeText          string  `json:"ptzTypeText"`
	HasAudio             bool    `json:"hasAudio"`
	SubCount             int     `json:"subCount"`
	Parental             int     `json:"parental"`
	ParentID             string  `json:"parentId"`
	Longitude            float64 `json:"longitude"`
	Latitude             float64 `json:"latitude"`
	CreateTime           string  `json:"createTime"`
	UpdateTime           string  `json:"updateTime"`
	StreamIdentification string  `json:"streamIdentification"`
}

func NewChannelView(ch *domainchannel.Channel) ChannelView {
	ptzText := ptzTypeTexts[ch.PTZType]
	if ptzText == "" {
		ptzText = ptzTypeTexts[0]
	}
	channelGBID := ch.GBDeviceID
	if channelGBID == "" {
		channelGBID = ch.DeviceID
	}
	streamID := ch.StreamIdentification
	if streamID == "" {
		streamID = "stream:0" // GB28181 目录不上报设备侧码流选择，默认主码流
	}
	return ChannelView{
		ID:                   ch.ID,
		InternalCode:         ch.InternalCode,
		ChannelID:            ch.ID,
		DeviceID:             channelGBID,
		ParentDeviceID:       ch.DeviceID,
		Name:                 ch.Name,
		Manufacturer:         ch.Manufacturer,
		Model:                ch.Model,
		Status:               ch.Status,
		PTZType:              ch.PTZType,
		PTZTypeText:          ptzText,
		HasAudio:             ch.HasAudio,
		SubCount:             ch.SubCount,
		Parental:             ch.Parental,
		ParentID:             ch.ParentID,
		Longitude:            ch.Longitude,
		Latitude:             ch.Latitude,
		CreateTime:           ch.CreateTime,
		UpdateTime:           ch.UpdateTime,
		StreamIdentification: streamID,
	}
}

func NewChannelViews(list []*domainchannel.Channel) []ChannelView {
	out := make([]ChannelView, 0, len(list))
	for _, ch := range list {
		if ch == nil {
			continue
		}
		out = append(out, NewChannelView(ch))
	}
	return out
}
