package publishauth

import "sync"

// PublishRegistry tracks streams that should enable MP4 recording on publish.
type PublishRegistry struct {
	mu     sync.RWMutex
	mp4Set map[string]struct{}
}

func NewPublishRegistry() *PublishRegistry {
	return &PublishRegistry{mp4Set: make(map[string]struct{})}
}

func streamKey(app, stream string) string { return app + "/" + stream }

func (r *PublishRegistry) EnableMP4(app, stream string) {
	r.mu.Lock()
	r.mp4Set[streamKey(app, stream)] = struct{}{}
	r.mu.Unlock()
}

func (r *PublishRegistry) DisableMP4(app, stream string) {
	r.mu.Lock()
	delete(r.mp4Set, streamKey(app, stream))
	r.mu.Unlock()
}

func (r *PublishRegistry) ShouldMP4(app, stream string) bool {
	r.mu.RLock()
	_, ok := r.mp4Set[streamKey(app, stream)]
	r.mu.RUnlock()
	return ok
}
