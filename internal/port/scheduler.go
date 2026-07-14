package port

import "errors"

var (
	ErrNoCandidate     = errors.New("no healthy candidate")
	ErrStoreDisabled   = errors.New("object store disabled or not configured")
	ErrStoreUnsupported = errors.New("object store provider not supported")
)

// LeastLoad 在健康候选中选负载最低者。
func LeastLoad[T LoadAware](candidates []T) (T, error) {
	var zero T
	var best T
	found := false
	var bestLoad int64
	for _, c := range candidates {
		if !c.Healthy() {
			continue
		}
		load := c.CurrentLoad()
		if !found || load < bestLoad {
			best = c
			bestLoad = load
			found = true
		}
	}
	if !found {
		return zero, ErrNoCandidate
	}
	return best, nil
}
