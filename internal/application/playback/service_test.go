package playbackapp

import (
	"sync"
	"testing"
)

func TestPlaybackNextSSRCSeqUniqueUnderConcurrency(t *testing.T) {
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
		set[v] = struct{}{}
	}
}

func TestParseDownloadSpeed(t *testing.T) {
	cases := map[string]int{"": 4, "0": 4, "-2": 4, "8": 8, "3": 3}
	for in, want := range cases {
		if got := ParseDownloadSpeed(in); got != want {
			t.Errorf("ParseDownloadSpeed(%q)=%d want %d", in, got, want)
		}
	}
}
