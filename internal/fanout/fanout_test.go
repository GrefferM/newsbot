package fanout

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestEachRunsEveryItemAndNoMoreThanAllowedAtOnce(t *testing.T) {
	var running, most atomic.Int32

	var mu sync.Mutex

	seen := map[int]int{}

	Each([]int{10, 20, 30, 40, 50, 60}, 2, func(i, item int) {
		now := running.Add(1)

		for {
			top := most.Load()
			if now <= top || most.CompareAndSwap(top, now) {
				break
			}
		}

		mu.Lock()
		seen[i] = item
		mu.Unlock()

		running.Add(-1)
	})

	if len(seen) != 6 || seen[5] != 60 || most.Load() > 2 {
		t.Errorf("seen %v, at once %d", seen, most.Load())
	}
}
