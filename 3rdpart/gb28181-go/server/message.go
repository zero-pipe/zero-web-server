package server

import (
	"context"
	"log"
	"time"

	"github.com/emiago/sipgo/sip"
	"github.com/zero-pipe/gb28181-go/manscdp"
)

func (s *Server) handleMessage(req *sip.Request, tx sip.ServerTransaction) {
	fromDeviceID := extractSIPUser(req)
	body := req.Body()
	msg, err := manscdp.Parse(body)
	if err != nil {
		s.respond(tx, req, 400, "Bad Request")
		return
	}
	deviceID := fromDeviceID
	if deviceID == "" {
		deviceID = msg.DeviceID
	}

	if s.handlers.Message == nil || !s.handlers.Message.DeviceKnown(deviceID) {
		s.respond(tx, req, 404, "Not Found")
		return
	}

	ctx := context.Background()
	switch msg.Root {
	case "Notify":
		switch msg.CmdType {
		case "Keepalive":
			ip, port := extractSourceAddr(req)
			_ = s.handlers.Message.OnKeepalive(ctx, deviceID, ip, port)
			if s.handlers.Telemetry != nil {
				s.handlers.Telemetry.OnKeepalive(deviceID, time.Now())
			}
		case "Catalog":
			log.Printf("[gb28181-go] catalog notify: device=%s items=%d", deviceID, len(msg.Items))
			_ = s.handlers.Message.OnCatalog(ctx, deviceID, msg.Items)
		case "Alarm":
			_ = s.handlers.Message.OnAlarm(ctx, deviceID, msg.DeviceID, msg.Alarm)
		case "MobilePosition":
			_ = s.handlers.Message.OnMobilePosition(ctx, deviceID, msg.DeviceID, msg.Position)
		}
	case "Response":
		switch msg.CmdType {
		case "Catalog":
			log.Printf("[gb28181-go] catalog response: device=%s items=%d", deviceID, len(msg.Items))
			_ = s.handlers.Message.OnCatalog(ctx, deviceID, msg.Items)
		case "DeviceInfo":
			devName := manscdp.ExtractTag(body, "DeviceName")
			if devName == "" {
				devName = manscdp.ExtractTag(body, "Name")
			}
			mfr := manscdp.ExtractTag(body, "Manufacturer")
			model := manscdp.ExtractTag(body, "Model")
			fw := manscdp.ExtractTag(body, "Firmware")
			log.Printf("[gb28181-go] DeviceInfo: device=%s name=%s manufacturer=%s model=%s",
				deviceID, devName, mfr, model)
			_ = s.handlers.Message.OnDeviceInfo(ctx, deviceID, devName, mfr, model, fw)
		case "RecordInfo":
			s.records.Handle(deviceID, msg.DeviceID, msg.SN, msg.SumNum, msg.RecordItems)
		case "PresetQuery":
			s.presets.Handle(msg.SN, msg.SumNum, msg.PresetItems)
		}
	}

	s.respond(tx, req, 200, "OK")
}
