package playapp

import (
	"sync"
	"testing"

	"zero-web-server/internal/port"
)

func TestNextSSRCSeqUniqueUnderConcurrency(t *testing.T) {
	s := &Service{}
	const n = 200
	var wg sync.WaitGroup
	seen := make([]int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			seen[i] = s.nextSSRCSeq()
		}(i)
	}
	wg.Wait()
	set := make(map[int]struct{}, n)
	for _, v := range seen {
		if _, dup := set[v]; dup {
			t.Fatalf("duplicate SSRC sequence %d", v)
		}
		if v <= 0 {
			t.Fatalf("sequence must be positive, got %d", v)
		}
		set[v] = struct{}{}
	}
	if len(set) != n {
		t.Fatalf("expected %d distinct sequences, got %d", n, len(set))
	}
}

func TestGBLiveStreamReady(t *testing.T) {
	cases := []struct {
		name string
		info *port.StreamProbe
		want bool
	}{
		{"nil", nil, false},
		{"no video", &port.StreamProbe{Video: false}, false},
		{"video low bytes", &port.StreamProbe{Video: true, BytesSpeed: 100}, false},
		{"video enough bytes", &port.StreamProbe{Video: true, BytesSpeed: 4096}, true},
		{"fps", &port.StreamProbe{Video: true, BytesSpeed: 10, VideoFps: 25}, true},
		{"resolution", &port.StreamProbe{Video: true, BytesSpeed: 10, Width: 640, Height: 480}, true},
	}
	for _, tc := range cases {
		if got := gbLiveStreamReady(tc.info); got != tc.want {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}
