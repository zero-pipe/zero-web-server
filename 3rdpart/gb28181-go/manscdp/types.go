package manscdp

// Message is a parsed MANSCDP XML document (Notify / Response / Query / Control).
type Message struct {
	Root        string
	CmdType     string
	SN          string
	DeviceID    string
	SumNum      int
	Raw         []byte
	Items       []CatalogItem
	RecordItems []RecordItem
	PresetItems []Preset
	Alarm       *AlarmNotify
	Position    *MobilePositionNotify
}

// CatalogItem is a device/channel entry in a Catalog response or notify.
type CatalogItem struct {
	DeviceID     string
	Name         string
	Manufacturer string
	Model        string
	Owner        string
	CivilCode    string
	Address      string
	Parental     int
	ParentID     string
	Status       string
	Longitude    float64
	Latitude     float64
	PTZType      int
}

// RecordItem is a record file entry in a RecordInfo response.
type RecordItem struct {
	DeviceID   string
	Name       string
	FilePath   string
	FileSize   string
	StartTime  string
	EndTime    string
	Secrecy    int
	Type       string
	RecorderID string
}

// Preset is a PTZ preset from PresetQuery.
type Preset struct {
	PresetID   string
	PresetName string
}

// AlarmNotify is an Alarm MANSCDP notify payload.
type AlarmNotify struct {
	AlarmPriority    string
	AlarmMethod      string
	AlarmTime        string
	AlarmDescription string
	Longitude        float64
	Latitude         float64
	AlarmType        int
}

// MobilePositionNotify is a MobilePosition MANSCDP notify payload.
type MobilePositionNotify struct {
	Longitude float64
	Latitude  float64
	Speed     float64
	Direction float64
	Altitude  float64
	Time      string
}
