package comparator

import (
	"sync"
	"sync/atomic"
)

// worker is a goroutine that receives unitPairs, compares them, and sends
// differing pairs to the results channel.
func worker(wg *sync.WaitGroup, jobs <-chan unitPair, results chan<- unitPair, threshold float64, processed *int64, comparator UnitComparator) {
	defer wg.Done()
	for pair := range jobs {
		diff := comparator.Compare(pair.UnitA.Img, pair.UnitB.Img)
		if diff > threshold {
			results <- pair
		}
		atomic.AddInt64(processed, 1)
	}
}

