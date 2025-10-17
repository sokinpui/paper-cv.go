package comparator

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// Run is the main application logic.
func Run(cfg *Config) error {
	log.Printf("Starting image comparison with %d CPU cores on device '%s'.", cfg.CPUCores, cfg.Device)
	runtime.GOMAXPROCS(cfg.CPUCores)

	comparator, err := NewUnitComparator(cfg.Device)
	if err != nil {
		return err
	}

	img, err := loadImage(cfg.InputPath, cfg.ImageType)
	if err != nil {
		return fmt.Errorf("failed to load image: %w", err)
	}

	units := splitImageIntoUnits(img, cfg.UnitSize)
	if len(units) < 2 {
		fmt.Println("Image resulted in fewer than two units. No comparison is possible.")
		return nil
	}
	fmt.Printf("Divided image into %d units.\n", len(units))

	jobs := make(chan unitPair, len(units))
	results := make(chan unitPair, len(units))

	var processedPairs int64
	totalPairs := int64(len(units) * (len(units) - 1) / 2)

	var wg sync.WaitGroup
	for i := 0; i < cfg.CPUCores; i++ {
		wg.Add(1)
		go worker(&wg, jobs, results, cfg.Threshold, &processedPairs, comparator)
	}

	var spinnerWg sync.WaitGroup
	spinnerWg.Add(1)
	done := make(chan struct{})
	go func() {
		defer spinnerWg.Done()
		s := spinner.New()
		s.Spinner = spinner.Dot
		s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				processed := atomic.LoadInt64(&processedPairs)
				fmt.Printf("\r%s Comparison complete. %d/%d pairs processed.\n", "✓", processed, totalPairs)
				return
			case <-ticker.C:
				s, _ = s.Update(spinner.TickMsg{})
				processed := atomic.LoadInt64(&processedPairs)
				fmt.Printf("\r%s Comparing units %d/%d...", s.View(), processed, totalPairs)
			}
		}
	}()

	log.Println("Generating and processing unit pairs...")
	startTime := time.Now()
	go func() {
		defer close(jobs)
		for i := 0; i < len(units); i++ {
			for j := i + 1; j < len(units); j++ {
				jobs <- unitPair{UnitA: units[i], UnitB: units[j]}
			}
		}
	}()

	wg.Wait()
	close(done)
	spinnerWg.Wait()

	duration := time.Since(startTime)
	log.Printf("Comparison of all units took %s.", duration)
	log.Printf("Comparisons per second: %.2f", float64(totalPairs)/duration.Seconds())

	durationStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("202"))
	speedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
	fmt.Printf("Total processing time: %s\n", durationStyle.Render(duration.String()))
	fmt.Printf("Comparisons per second: %s\n", speedStyle.Render(fmt.Sprintf("%.2f", float64(totalPairs)/duration.Seconds())))

	close(results)

	log.Println("Saving differing pairs...")
	saveDifferentPairs(results, cfg.OutputDirectory)
	log.Println("Processing complete.")
	return nil
}
