package syncgroup

import (
	"sync"
	"sync/atomic"
)

// SyncGroup acts as a sync.WaitGroup which allows goroutes to Sync on one another.
// The use case it was designed for batching, where you wish to accumulate something until it
// reaches a certain size or time (tracked via the batcher), or if work is blocked on the batching.
type SyncGroup struct {
	queued  int32
	stopped int32
	paused  int32

	finished *sync.Cond
	once     sync.Once
}

// Add sets how many jobs are being queued up
func (sg *SyncGroup) Add(j int) {
	sg.once.Do(func() {
		sg.finished = sync.NewCond(&sync.Mutex{})
	})

	sg.finished.L.Lock()
	atomic.AddInt32(&sg.queued, int32(j))
	sg.finished.L.Unlock()
}

// Go runs a function in a goroutine, tracking it's execution state in the SyncGroup
// The supplied function can call (*SyncGroup).Sync() to pause execution until all
// tracked goroutines are also in a call to Sync()
func (sg *SyncGroup) Go(fn func()) {
	if sg.finished.L == nil {
		panic("Cannot call SyncGroup.Go before SyncGroup.Add")
	}

	go func() {
		defer func() {
			sg.finished.L.Lock()
			atomic.AddInt32(&sg.stopped, 1)
			sg.finished.L.Unlock()
			sg.finished.Broadcast()
		}()

		fn()
	}()
}

// Wait will block until all tracked goroutines finish executing
func (sg *SyncGroup) Wait() {
	for {
		sg.finished.L.Lock()
		if atomic.LoadInt32(&sg.queued) == atomic.LoadInt32(&sg.stopped) {
			sg.finished.L.Unlock()
			return
		}
		sg.finished.Wait()
		sg.finished.L.Unlock()
	}
}

// Sync will block until all goroutines are either finished or waiting on Sync
func (sg *SyncGroup) Sync() {
	sg.finished.L.Lock()
	atomic.AddInt32(&sg.paused, 1)
	sg.finished.L.Unlock()
	sg.finished.Broadcast()

	for {
		sg.finished.L.Lock()
		if sg.done() {
			atomic.AddInt32(&sg.paused, -1)
			sg.finished.L.Unlock()
			return
		}
		sg.finished.Wait()
		sg.finished.L.Unlock()
	}
}

func (sg *SyncGroup) done() bool {
	if atomic.LoadInt32(&sg.queued) == (atomic.LoadInt32(&sg.stopped) + atomic.LoadInt32(&sg.paused)) {
		return true
	}
	return false
}
