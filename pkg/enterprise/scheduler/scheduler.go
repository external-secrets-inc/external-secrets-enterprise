package scheduler

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Scheduler interface {
	// ScheduleInterval schedules a function to run every d time.
	ScheduleInterval(key string, interval, timeout time.Duration, fn func(context.Context, logr.Logger))

	// Cancel cancels a scheduled function
	Cancel(key string)

	Start(ctx context.Context) error
}

type job struct {
	stop     context.CancelFunc
	duration time.Duration
}

type SchedulerImpl struct {
	log    logr.Logger
	ctx    context.Context
	mu     sync.Mutex
	leader atomic.Bool
	jobs   map[string]job
	client client.Client
}

func New(client client.Client, log logr.Logger) Scheduler {
	return &SchedulerImpl{
		jobs:   map[string]job{},
		client: client,
		log:    log,
	}
}

func (s *SchedulerImpl) ScheduleInterval(key string, interval, timeout time.Duration, fn func(context.Context, logr.Logger)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if e, ok := s.jobs[key]; ok {
		e.stop()
	}

	ctx, cancel := context.WithCancel(s.ctx)
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				s.runWithTimeout(fn, timeout)
			case <-ctx.Done():
				return
			}
		}
	}()

	s.log.Info("Scheduled job", "key", key, "interval", interval)
	s.jobs[key] = job{stop: cancel, duration: interval}
}

func (s *SchedulerImpl) Cancel(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.jobs[key]; ok {
		e.stop()
		delete(s.jobs, key)
	}
	s.log.Info("Canceled job", "key", key)
}

func (s *SchedulerImpl) NeedLeaderElection() bool { return true }

func (s *SchedulerImpl) Start(ctx context.Context) error {
	s.log.Info("Starting scheduler")
	s.leader.Store(true)
	s.ctx = ctx
	defer func() {
		s.leader.Store(false)
		for _, e := range s.jobs {
			e.stop()
		}
	}()

	<-ctx.Done()
	return nil
}

func (s *SchedulerImpl) IsLeader() bool { return s.leader.Load() }

func (s *SchedulerImpl) runWithTimeout(fn func(ctx context.Context, log logr.Logger), max time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), max)
	defer cancel()
	fn(ctx, s.log)
}
