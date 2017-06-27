package syncgroup

import (
	"fmt"
	"sync"
)

func ExampleSyncGroup() {
	// It is expected that SyncGroup is used in libraries
	// We represent that here with the randomFetcher struct
	f := &randomFetcher{}
	f.Add(10)
	for i := 0; i < 10; i++ {
		// The library author may require end-users to manually use SyncGroup.Go
		// or may offer a utility method to abstract this away.
		// Here the user does it manually.
		f.Go(func() {
			// Usage of SyncGroup.Sync is abstracted away into the library's methods
			fmt.Println(f.Rand())
		})
	}

	// The level of abstraction for SyncGroup.Wait is expected to match that of .Go
	f.Wait()

	// Unordered Output:
	// Fetching 10 random numbers...
	// 0
	// 1
	// 2
	// 3
	// 4
	// 5
	// 6
	// 7
	// 8
	// 9
}

type randomFetcher struct {
	SyncGroup
	sync.Mutex
	count int
	rands []int
}

// Rand returns a "random" number
func (f *randomFetcher) Rand() int {
	// Signal that we're waiting for a random number
	f.Lock()
	f.count++
	f.Unlock()
	// Wait for batching to complete
	f.Sync()

	f.Lock()         // Race for the lock
	if f.count > 0 { // Whoever is first, fetch the random numbers
		f.getRands()
	}
	i := f.rands[0]       // Get a rand out of the populated pool
	f.rands = f.rands[1:] // Reduce the pool
	f.Unlock()

	return i
}

// getRands assumes it is called within a lock
func (f *randomFetcher) getRands() {
	// Here we could do some expensive operation like make an HTTP request.
	// Instead we just use a constant list so the example functions
	fmt.Printf("Fetching %d random numbers...\n", f.count)
	f.count = 0
	f.rands = []int{8, 6, 5, 9, 1, 3, 4, 2, 7, 0}
}
