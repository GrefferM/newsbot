package fanout

import "sync"

func Each[T any](items []T, atOnce int, do func(i int, item T)) {
	slots := make(chan struct{}, max(atOnce, 1))

	var wg sync.WaitGroup

	for i, item := range items {
		wg.Go(func() {
			slots <- struct{}{}

			defer func() {
				<-slots
			}()

			do(i, item)
		})
	}

	wg.Wait()
}
