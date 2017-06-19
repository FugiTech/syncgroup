package syncgroup

import "sync"

// SyncGroup is a sync.WaitGroup that also allows tracked goroutines to "sync" their execution.
// The use case it was designed for batching, where you wish to accumulate something until it
// reaches a certain size or time (tracked via the batcher), or if work is blocked on the batching.
type SyncGroup struct {
	running  sync.WaitGroup
	syncing  sync.WaitGroup
	finished sync.RWMutex

	lock  sync.Mutex
	count int
}

// Go runs a function in a goroutine, tracking it's execution state in the SyncGroup
// The supplied function can call (*SyncGroup).Sync() to pause execution until all
// tracked goroutines are also in a call to Sync()
func (sg *SyncGroup) Go(fn func()) {
	sg.lock.Lock()
	defer sg.lock.Unlock()

	sg.running.Add(1) // Must be called before sg.watch or it will immediately terminate
	if sg.count == 0 {
		go sg.watch()
	}
	sg.count++

	go func() {
		defer func() {
			sg.lock.Lock()
			defer sg.lock.Unlock()
			sg.count-- // Must be called before calling sg.running.Done() or sg.watch() will break
			sg.running.Done()
		}()

		fn()
	}()
}

// Wait will block until all tracked goroutines finish executing
func (sg *SyncGroup) Wait() {
	sg.finished.RLock()
	sg.finished.RUnlock()
}

// Sync will block execution until all other tracked goroutines call Sync() or terminate
func (sg *SyncGroup) Sync() {
	sg.syncing.Add(1)
	sg.running.Done()
	sg.syncing.Wait()
}

// Watch maintains the (*SyncGroup).finished lock (used in Wait) and
// calls Done on the syncing WaitGroup when deadlock is reached
func (sg *SyncGroup) watch() {
	sg.finished.Lock()
	defer sg.finished.Unlock()

	for {
		sg.running.Wait() // Wait for all goroutines to Sync() or terminate
		// Theoretically there's a race condition here where you could call Go() between Wait() and Lock() running...
		// But I don't know how to get around that
		// If it _does_ happen, the calls to Add() below will have the wrong count and will likely panic
		sg.lock.Lock()
		if sg.count == 0 { // All done!
			sg.lock.Unlock()
			return
		}

		sg.running.Add(sg.count)
		sg.syncing.Add(-1 * sg.count)
		sg.lock.Unlock()
	}
}
