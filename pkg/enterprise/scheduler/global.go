package scheduler

import (
	"sync"
)

var (
	global *Scheduler
	once   sync.Once
)

func SetGlobal(s *Scheduler) {
	if s == nil {
		panic("scheduler: SetGlobal called with nil")
	}
	once.Do(func() {
		global = s
	})
	if global != s {
		panic("scheduler: SetGlobal called more than once")
	}
}

func Global() Scheduler {
	if global == nil {
		panic("scheduler: Global called before SetGlobal")
	}
	return *global
}
