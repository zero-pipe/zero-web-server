package record

type RecordInfo struct {
	DeviceID   string       `json:"deviceId"`
	ChannelID  string       `json:"channelId"`
	SN         string       `json:"sn"`
	Name       string       `json:"name"`
	SumNum     int          `json:"sumNum"`
	Count      int          `json:"count"`
	RecordList []RecordItem `json:"recordList"`
}

type RecordItem struct {
	DeviceID   string `json:"deviceId"`
	Name       string `json:"name"`
	FilePath   string `json:"filePath"`
	FileSize   string `json:"fileSize"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
	Secrecy    int    `json:"secrecy"`
	Type       string `json:"type"`
	RecorderID string `json:"recorderId"`
}
