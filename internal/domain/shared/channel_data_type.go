package shared

const (
	ChannelDataTypeGB28181     = 1
	ChannelDataTypeStreamPush  = 2
	ChannelDataTypeStreamProxy = 3
	ChannelDataTypeONVIF       = 4
	ChannelDataTypeJT1078      = 200
)

func ChannelDataTypeDesc(dataType int) string {
	switch dataType {
	case ChannelDataTypeGB28181:
		return "国标28181"
	case ChannelDataTypeStreamPush:
		return "推流设备"
	case ChannelDataTypeStreamProxy:
		return "拉流代理"
	case ChannelDataTypeONVIF:
		return "ONVIF设备"
	case ChannelDataTypeJT1078:
		return "部标设备"
	default:
		return "未知"
	}
}
