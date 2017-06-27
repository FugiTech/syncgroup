package syncgroup_test

import (
	"os"
	"sync/atomic"
	"testing"

	"github.com/ossareh/syncgroup"
)

func iterCount(v int) int {
	if e := os.Getenv("RACE_TEST"); e != "" {
		return 30
	}

	return v
}

func TestSyncGroup_Wait(t *testing.T) {
	// In the most basic case SyncGroup should behave like a (*sync.WaitGroup)
	iters := iterCount(2000000)
	decrements := int32(iters)

	sg := &syncgroup.SyncGroup{}
	sg.Add(iters)

	for i := 0; i < iters; i++ {
		sg.Go(func() {
			atomic.AddInt32(&decrements, -1)
		})
	}
	sg.Wait()

	if atomic.LoadInt32(&decrements) != 0 {
		t.Errorf("TestSyncGroup_Wait expected=0 got=%d", decrements)
	}
}

func TestSyncGroup_Sync(t *testing.T) {
	// The real purpose of SyncGroup is to allow functions to Sync() on eachother
	iters := iterCount(200000)
	decrements := int32(iters)
	increments := int32(0)

	sg := &syncgroup.SyncGroup{}
	sg.Add(iters)

	for i := 0; i < iters; i++ {
		sg.Go(func() {
			atomic.AddInt32(&decrements, -1)
			sg.Sync()

			if atomic.LoadInt32(&decrements) != 0 {
				t.Errorf("TestSyncGroup_Sync goroutine expected=0 got=%d", decrements)
			}
			atomic.AddInt32(&increments, 2)
		})
	}
	sg.Wait()

	if atomic.LoadInt32(&increments) != int32(iters*2) {
		t.Errorf("TestSyncGroup_Sync expected=%d got=%d", iters*2, increments)
	}
}

func TestSyncGroup_Mixed(t *testing.T) {
	// Covers the case of mixed function types, some that sync and some that don't
	iters := iterCount(2000000)
	decrements := int32(iters)
	increments := int32(0)

	sg := &syncgroup.SyncGroup{}
	sg.Add(iters)

	for i := 0; i < iters; i++ {
		if i%10 == 0 {
			sg.Go(func() {
				atomic.AddInt32(&decrements, -1)
				sg.Sync()

				if atomic.LoadInt32(&decrements) != 0 {
					t.Errorf("TestSyncGroup_Mixed goroutine expected=0 got=%d", decrements)
				}
				atomic.AddInt32(&increments, 2)
			})
		} else {
			sg.Go(func() {
				atomic.AddInt32(&decrements, -1)
			})
		}
	}

	sg.Wait()

	expected := int(iters/10) * 2
	if atomic.LoadInt32(&increments) != int32(expected) {
		t.Errorf("TestSyncGroup_Sync expected=%d got=%d", expected, increments)
	}
}

func TestSyncGroup_MultiAdd(t *testing.T) {
	// Add many batches
	iters := iterCount(1000000)
	batches := iters / 10
	decrements := int32(iters)
	increments := int32(0)

	sg := &syncgroup.SyncGroup{}

	for j := 0; j < batches; j++ {
		sg.Add(iters / batches)

		for i := 0; i < iters/batches; i++ {
			if i%10 == 0 {
				sg.Go(func() {
					atomic.AddInt32(&decrements, -1)
					sg.Sync()

					if atomic.LoadInt32(&decrements) != 0 {
						t.Errorf("TestSyncGroup_MultiAdd goroutine expected=0 got=%d", decrements)
					}
					atomic.AddInt32(&increments, 2)
				})
			} else {
				sg.Go(func() {
					atomic.AddInt32(&decrements, -1)
				})
			}
		}
	}

	sg.Wait()

	expected := int(iters/10) * 2
	if atomic.LoadInt32(&increments) != int32(expected) {
		t.Errorf("TestSyncGroup_Sync expected=%d got=%d", expected, increments)
	}
}
