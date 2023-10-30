package simulation

import "sync"

func parallelFor(start, end int, f func(int)) {
	tasks := make(chan int, end-start)
	var wg sync.WaitGroup

	// Number of workers, could be configurable
	numWorkers := 4

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range tasks {
				f(i)
			}
		}()
	}

	// Feed tasks
	for i := start; i < end; i++ {
		tasks <- i
	}

	// Close channel and wait for workers
	close(tasks)
	wg.Wait()
}
