// Package discovery keeps the applier's REST mapper cache current.
package discovery

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
)

// Refresher invalidates cached discovery immediately and on a fixed interval
// so newly installed resource types become visible without restarting the applier.
type Refresher struct {
	mapper   meta.ResettableRESTMapper
	interval time.Duration
}

// New builds a Refresher for mapper. interval is the time between resets.
func New(mapper meta.ResettableRESTMapper, interval time.Duration) *Refresher {
	return &Refresher{mapper: mapper, interval: interval}
}

// Start resets the mapper immediately, then on every tick, until ctx is canceled.
// Caller-driven cancellation returns nil.
func (r *Refresher) Start(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		r.mapper.Reset()
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
