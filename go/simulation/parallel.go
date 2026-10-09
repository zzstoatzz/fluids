package simulation

import (
	"runtime"
	"sync"
)

// ParallelConfig holds configuration for parallel execution
type ParallelConfig struct {
	// NumWorkers is the number of goroutines to use for parallel execution
	// If <= 0, it will use runtime.NumCPU() 
	NumWorkers int
	
	// MinimumBatchSize is the minimum number of items that should be processed in parallel
	// For very small batches, serial execution may be faster
	MinimumBatchSize int
}

// defaultParallelConfig provides reasonable defaults
var defaultParallelConfig = ParallelConfig{
	NumWorkers:       runtime.NumCPU(),
	MinimumBatchSize: 32,
}

// SetParallelConfig allows changing the global parallel processing configuration
func SetParallelConfig(config ParallelConfig) {
	if config.NumWorkers > 0 {
		defaultParallelConfig.NumWorkers = config.NumWorkers
	}
	if config.MinimumBatchSize > 0 {
		defaultParallelConfig.MinimumBatchSize = config.MinimumBatchSize
	}
}

// parallelFor executes the given function in parallel for values from start to end-1
func parallelFor(start, end int, f func(int)) {
	// If batch is small enough, just run serially
	if end-start <= defaultParallelConfig.MinimumBatchSize {
		for i := start; i < end; i++ {
			f(i)
		}
		return
	}
	
	// Determine number of workers
	numWorkers := defaultParallelConfig.NumWorkers
	
	// For very small workloads, limit the number of goroutines
	if itemsPerWorker := (end - start) / numWorkers; itemsPerWorker < 4 {
		numWorkers = (end - start + 3) / 4 // Ensure at least 4 items per worker
	}
	
	// No point having more workers than items
	if numWorkers > (end - start) {
		numWorkers = end - start
	}
	
	// If we ended up with just 1 worker, run serially
	if numWorkers <= 1 {
		for i := start; i < end; i++ {
			f(i)
		}
		return
	}
	
	// Use a worker pool pattern with chunking for better performance
	var wg sync.WaitGroup
	itemsPerWorker := (end - start + numWorkers - 1) / numWorkers
	
	for workerIdx := 0; workerIdx < numWorkers; workerIdx++ {
		wg.Add(1)
		go func(workerIdx int) {
			defer wg.Done()
			
			// Calculate chunk bounds for this worker
			chunkStart := start + (workerIdx * itemsPerWorker)
			chunkEnd := chunkStart + itemsPerWorker
			if chunkEnd > end {
				chunkEnd = end
			}
			
			// Process assigned chunk
			for i := chunkStart; i < chunkEnd; i++ {
				f(i)
			}
		}(workerIdx)
	}
	
	wg.Wait()
}
