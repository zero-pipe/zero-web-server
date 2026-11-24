package onvifapp

import (
	"strings"
	"testing"
	"time"

	domainonvif "zero-web-server/internal/domain/onvif"

	onviflib "zero-web-server/3rdpart/onvif-go"
)

// 主题映射是 ONVIF 事件唯一会"静默出错"的地方：映射错了一整类告警就归错类，
// 映射漏了则完全不出事件。这些用例锁住常见主题的判定。
func TestMapONVIFTopic(t *testing.T) {
	cases := []struct {
		topic string
		want  string
	}{
		{"tns1:RuleEngine/CellMotionDetector/Motion", "motion"},
		{"tns1:VideoSource/MotionAlarm", "motion"},
		{"tns1:RuleEngine/LineDetector/Crossed", "line_crossing"},
		{"tns1:RuleEngine/FieldDetector/ObjectsInside", "intrusion"},
		{"tns1:RuleEngine/LoiterDetector/Loitering", "loitering"},
		{"tns1:VideoSource/ImageTooBlurry/AnalyticsService", "scene_change"},
		{"tns1:VideoSource/GlobalSceneChange/ImagingService", "scene_change"},
		{"tns1:VideoSource/VideoLoss/Processor", "video_loss"},
		{"tns1:AudioAnalytics/Audio/DetectedSound", "audio_abnormal"},
		{"tns1:RuleEngine/MyRuleDetector/FaceDetect", "face"},
		{"tns1:RuleEngine/MyRuleDetector/VehiclePlate", "plate"},
		{"tns1:Device/Trigger/DigitalInput", "alarm_input"},
		{"tns1:PTZController/PTZConfiguration/Move", "ptz"},
		// 存储故障归"录像异常"：对用户来说就是录不上了。
		{"tns1:Device/HardwareFailure/StorageFailure", "record_abnormal"},
		{"tns1:Unknown/Whatever", ""},
	}
	for _, c := range cases {
		if got := mapONVIFTopic(c.topic); got != c.want {
			t.Errorf("mapONVIFTopic(%q) = %q, want %q", c.topic, got, c.want)
		}
	}
}

// 越界/入侵必须优先于泛化的 Motion 规则命中。
func TestMapONVIFTopicSpecificBeatsGeneric(t *testing.T) {
	for _, topic := range []string{
		"tns1:RuleEngine/LineDetector/Crossed",
		"tns1:RuleEngine/FieldDetector/ObjectsInside",
	} {
		if got := mapONVIFTopic(topic); got == "motion" {
			t.Errorf("%q 被泛化的 motion 规则吃掉，得到 %q", topic, got)
		}
	}
}

func TestNotificationActive(t *testing.T) {
	cases := []struct {
		name string
		msg  onviflib.EventMessage
		want bool
	}{
		{"无状态字段按触发处理", onviflib.EventMessage{}, true},
		{"State=true", onviflib.EventMessage{
			Data: []onviflib.SimpleItem{{Name: "State", Value: "true"}},
		}, true},
		{"State=false 是恢复，不投递", onviflib.EventMessage{
			Data: []onviflib.SimpleItem{{Name: "State", Value: "false"}},
		}, false},
		{"LogicalState=false", onviflib.EventMessage{
			Data: []onviflib.SimpleItem{{Name: "LogicalState", Value: "false"}},
		}, false},
		{"IsMotion=false", onviflib.EventMessage{
			Data: []onviflib.SimpleItem{{Name: "IsMotion", Value: "false"}},
		}, false},
		{"非布尔 Value 视为触发", onviflib.EventMessage{
			Data: []onviflib.SimpleItem{{Name: "Value", Value: "3"}},
		}, true},
	}
	for _, c := range cases {
		got := notificationActive(onviflib.NotificationMessage{Message: c.msg})
		if got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestOnvifEventIDStableAndUnique(t *testing.T) {
	dev := &domainonvif.Device{InternalCode: "D001"}
	at := time.Unix(1700000000, 123)
	a := onvifEventID(dev, "tns1:RuleEngine/CellMotionDetector/Motion", at)
	b := onvifEventID(dev, "tns1:RuleEngine/CellMotionDetector/Motion", at)
	if a != b {
		t.Errorf("同一事件两次生成 ID 不一致: %s vs %s", a, b)
	}
	c := onvifEventID(dev, "tns1:RuleEngine/CellMotionDetector/Motion", at.Add(time.Millisecond))
	if a == c {
		t.Errorf("不同时刻的事件 ID 相同: %s", a)
	}
	if !strings.HasPrefix(a, "onvif-") {
		t.Errorf("ID 前缀不对: %s", a)
	}
}
