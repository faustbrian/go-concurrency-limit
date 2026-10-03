package concurrencylimit_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	concurrencylimit "github.com/faustbrian/go-concurrency-limit"
)

func TestConcurrentAcquireCompleteSnapshotAndResetRemainBounded(t *testing.T) {
	limiter, err := concurrencylimit.New(concurrencylimit.Config{
		MinLimit: 1, MaxLimit: 8, InitialLimit: 4,
		Algorithm: concurrencylimit.NewFixedAlgorithm(),
		Queue:     concurrencylimit.QueueConfig{MaxQueued: 64, MaxWait: time.Second},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	permit, err := limiter.Acquire(ctx)
	cancel()
	if err != nil || permit == nil {
		t.Fatalf("uncontended admission = permit %v, error %v", permit, err)
	}
	if snapshot := limiter.Snapshot(); snapshot.InFlight != 1 || snapshot.Queued != 0 {
		t.Fatalf("uncontended admission state = %+v", snapshot)
	}
	if err = permit.Complete(concurrencylimit.OutcomeIgnored); err != nil {
		t.Fatal(err)
	}
	if snapshot := limiter.Snapshot(); snapshot.InFlight != 0 || snapshot.Queued != 0 || snapshot.Samples != 0 {
		t.Fatalf("uncontended completion state = %+v", snapshot)
	}

	const workers = 32
	var wait sync.WaitGroup
	wait.Add(workers)
	for worker := range workers {
		go func() {
			defer wait.Done()
			for iteration := range 100 {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				permit, acquireErr := limiter.Acquire(ctx)
				cancel()
				if acquireErr != nil {
					if errors.Is(acquireErr, concurrencylimit.ErrReset) || errors.Is(acquireErr, context.DeadlineExceeded) {
						continue
					}
					t.Errorf("worker %d iteration %d: Acquire() error = %v", worker, iteration, acquireErr)
					return
				}
				_ = limiter.Snapshot()
				completionErr := permit.Complete(concurrencylimit.Outcome(iteration % 5))
				if completionErr != nil && !errors.Is(completionErr, concurrencylimit.ErrStalePermit) {
					t.Errorf("worker %d iteration %d: Complete() error = %v", worker, iteration, completionErr)
					return
				}
			}
		}()
	}
	for range 10 {
		time.Sleep(time.Millisecond)
		limiter.Reset()
	}
	wait.Wait()
	snapshot := limiter.Snapshot()
	if snapshot.Limit < 1 || snapshot.Limit > 8 || snapshot.InFlight != 0 || snapshot.Queued != 0 {
		t.Fatalf("final Snapshot() = %+v", snapshot)
	}
}

func TestCanceledWaitersTerminateWithoutBackgroundWorkers(t *testing.T) {
	t.Parallel()

	limiter := newFixedLimiter(t, 1, concurrencylimit.QueueConfig{MaxQueued: 32, MaxWait: time.Second})
	active, err := limiter.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	wait.Add(32)
	for range 32 {
		go func() {
			defer wait.Done()
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, _ = limiter.Acquire(ctx)
		}()
	}
	done := make(chan struct{})
	go func() { wait.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("canceled acquisition goroutines did not terminate")
	}
	if err = active.Complete(concurrencylimit.OutcomeSuccess); err != nil {
		t.Fatal(err)
	}
}
