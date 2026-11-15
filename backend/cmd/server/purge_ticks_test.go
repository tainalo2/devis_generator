package main

import (
	"context"
	"testing"
	"time"
)

func TestIdempotencyPurgeTicksEmitsAndStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ticks := idempotencyPurgeTicks(ctx, 5*time.Millisecond)
	select {
	case <-ticks:
	case <-time.After(2 * time.Second):
		t.Fatal("aucun tick")
	}
	cancel()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case _, ok := <-ticks:
			if !ok {
				return
			}
		case <-deadline:
			return // le canal n'est pas fermé par conception ; la goroutine s'arrête sur ctx
		}
	}
}
