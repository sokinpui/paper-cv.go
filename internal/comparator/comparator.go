package comparator

import (
	"fmt"
	"log"
	"runtime"
	"sync"
)

// Run is the main application logic.
func Run(cfg *Config) error {
	log.Printf("Starting image comparison with %d CPU cores.", cfg.CPUCores)
	runtime.GOMAXPROCS(cfg.CPUCores)

	img, err := loadImage(cfg.InputPath, cfg.ImageType)
	if err != nil {
		return fmt.Errorf("failed to load image: %w", err)
	}

	units := splitImageIntoUnits(img, cfg.UnitSize)
	if len(units) < 2 {
		log.Println("Image resulted in fewer than two units. No comparison is possible.")
		return nil
	}
	log.Printf("Divided image into %d units.", len(units))

	// A channel to send pairs of units to worker goroutines.
	jobs := make(chan unitPair, len(units))
	// A channel to receive pairs that are found to be different.
	results := make(chan unitPair, len(units))

	var wg sync.WaitGroup
	for i := 0; i < cfg.CPUCores; i++ {
		wg.Add(1)
		go worker(&wg, jobs, results, cfg.Threshold)
	}

	// A separate goroutine to handle saving results to avoid contention.
	var saveWg sync.WaitGroup
	saveWg.Add(1)
	go func() {
		defer saveWg.Done()
		saveDifferentPairs(results, cfg.OutputDirectory)
	}()

	// Generate all unique pairs and send them to the jobs channel.
	log.Println("Generating and processing unit pairs...")
	for i := 0; i < len(units); i++ {
		for j := i + 1; j < len(units); j++ {
			jobs <- unitPair{UnitA: units[i], UnitB: units[j]}
		}
	}
	close(jobs)

	wg.Wait()
	close(results)
	saveWg.Wait()

	log.Println("Processing complete.")
	return nil
}
