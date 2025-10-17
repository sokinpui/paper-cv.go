package comparator

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"
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

	// Generate all unique pairs and send them to the jobs channel.
	log.Println("Generating and processing unit pairs...")
	startTime := time.Now()
	for i := 0; i < len(units); i++ {
		for j := i + 1; j < len(units); j++ {
			jobs <- unitPair{UnitA: units[i], UnitB: units[j]}
		}
	}
	close(jobs)
	wg.Wait()

	duration := time.Since(startTime)
	log.Printf("Comparison of all units took %s.", duration)

	close(results)

	log.Println("Saving differing pairs...")
	saveDifferentPairs(results, cfg.OutputDirectory)
	log.Println("Processing complete.")
	return nil
}
