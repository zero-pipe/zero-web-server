package dto

import domainchannel "zero-web-server/internal/domain/channel"

var ptzTypeTexts = map[int]string{
	0: "未知",
	1: "球机",
	2: "半球",
	3: "固定枪机",
	4: "遥控枪机",
}

type ChannelView struct {
	ID                   int     `json:"id"`
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
	return ChannelView{
		ID:             ch.ID,
		ChannelID:      ch.ID,
		DeviceID:       channelGBID,
		ParentDeviceID: ch.DeviceID,
		Name:           ch.Name,
		Manufacturer:   ch.Manufacturer,
		Model:          ch.Model,
		Status:         ch.Status,
		PTZType:        ch.PTZType,
		PTZTypeText:    ptzText,
		HasAudio:       ch.HasAudio,
		SubCount:       ch.SubCount,
		Parental:       ch.Parental,
		ParentID:       ch.ParentID,
		Longitude:      ch.Longitude,
		Latitude:       ch.Latitude,
		CreateTime:     ch.CreateTime,
		UpdateTime:     ch.UpdateTime,
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
