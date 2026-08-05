package sipinfra

import "github.com/zero-pipe/gb28181-go/manscdp"

func BuildSubscribeCatalog(deviceID, sn string) string {
	return manscdp.BuildSubscribeCatalog(deviceID, sn)
}

func BuildSubscribeAlarm(deviceID, sn string) string {
	return manscdp.BuildSubscribeAlarm(deviceID, sn)
}

func BuildSubscribeMobilePosition(deviceID, sn string, interval int) string {
	return manscdp.BuildSubscribeMobilePosition(deviceID, sn, interval)
}
