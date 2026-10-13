package discovery

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
)

type countingResettableMapper struct {
	meta.RESTMapper
	resets atomic.Int32
}

func (m *countingResettableMapper) Reset() {
	m.resets.Add(1)
}

func TestRefresher_RefreshesPeriodicallyAndStopsCleanly(t *testing.T) {
	mapper := &countingResettableMapper{}
	refresher := New(mapper, 5*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- refresher.Start(ctx)
	}()

	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for mapper.resets.Load() < 2 {
		select {
		case <-ticker.C:
		case <-deadline.C:
			cancel()
			t.Fatal("discovery mapper was not refreshed periodically")
		}
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Start() error = %v, want nil after cancellation", err)
		}
	case <-time.After(time.Second):
		t.Fatal("discovery refresher did not stop after cancellation")
	}
}
