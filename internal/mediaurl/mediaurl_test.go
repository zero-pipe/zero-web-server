package mediaurl

import "testing"

func TestLiveStreamRequiresBothCodes(t *testing.T) {
	cases := []struct {
		device  string
		channel string
		want    string
	}{
		{"D0000000000000001", "C0000000000000001", "D0000000000000001_C0000000000000001"},
		{" D1 ", " C1 ", "D1_C1"},
		{"", "C1", ""},
		{"D1", "", ""},
		{" ", " ", ""},
	}
	for _, tc := range cases {
		if got := LiveStream(tc.device, tc.channel); got != tc.want {
			t.Errorf("LiveStream(%q,%q)=%q want %q", tc.device, tc.channel, got, tc.want)
		}
	}
}

func TestPlaybackAndDownloadStream(t *testing.T) {
	if got := PlaybackStream("D1", "C1"); got != "D1_C1_playback" {
		t.Fatalf("PlaybackStream=%q", got)
	}
	if got := DownloadStream("D1", "C1"); got != "D1_C1_download" {
		t.Fatalf("DownloadStream=%q", got)
	}
	if got := PlaybackStream("", "C1"); got != "" {
		t.Fatalf("PlaybackStream with missing device=%q", got)
	}
	if got := DownloadStream("D1", ""); got != "" {
		t.Fatalf("DownloadStream with missing channel=%q", got)
	}
}
