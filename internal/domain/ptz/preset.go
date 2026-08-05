package ptz

// Preset is a camera PTZ preset reported by the device (GB28181 or ONVIF).
type Preset struct {
	PresetID   string `json:"presetId"`
	PresetName string `json:"presetName"`
}
