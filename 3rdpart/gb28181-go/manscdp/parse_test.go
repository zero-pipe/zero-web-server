package manscdp

import "testing"

func TestParsePresetItems(t *testing.T) {
	body := []byte(`<?xml version="1.0"?>
<Response>
<CmdType>PresetQuery</CmdType>
<SN>12</SN>
<DeviceID>34020000001320000001</DeviceID>
<SumNum>2</SumNum>
<PresetList Num="2">
<Item>
<PresetID>1</PresetID>
<PresetName>门口</PresetName>
</Item>
<Item>
<PresetID>2</PresetID>
<PresetName>大厅</PresetName>
</Item>
</PresetList>
</Response>`)
	msg, err := Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	if msg.CmdType != "PresetQuery" {
		t.Fatalf("CmdType=%s", msg.CmdType)
	}
	if msg.SumNum != 2 {
		t.Fatalf("SumNum=%d", msg.SumNum)
	}
	if len(msg.PresetItems) != 2 {
		t.Fatalf("items=%d", len(msg.PresetItems))
	}
	if msg.PresetItems[0].PresetID != "1" || msg.PresetItems[0].PresetName != "门口" {
		t.Fatalf("item0=%+v", msg.PresetItems[0])
	}
}

func TestParsePresetEmptyList(t *testing.T) {
	body := []byte(`<Response>
<CmdType>PresetQuery</CmdType>
<SN>1</SN>
<DeviceID>34020000001320000001</DeviceID>
<SumNum>0</SumNum>
<PresetList Num="0">
</PresetList>
</Response>`)
	msg, err := Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(msg.PresetItems) != 0 {
		t.Fatalf("expected empty, got %+v", msg.PresetItems)
	}
}
