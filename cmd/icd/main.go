package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/spf13/pflag"
)

// Config holds all the configuration parameters for the application,
// parsed from command-line flags.
type Config struct {
	InputPath       string
	OutputDirectory string
	UnitSize        int
	Threshold       float64
	CPUCores        int
}

// Unit represents a single square subdivision of the source image.
// It holds the image data and its coordinate position in the grid.
type Unit struct {
	ID  image.Point
	Img image.Image
}

// UnitPair represents a pair of Units that need to be compared.
type UnitPair struct {
	UnitA *Unit
	UnitB *Unit
}

func main() {
	cfg := parseFlags()
	if err := validateConfig(cfg); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	if err := run(cfg); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}

// parseFlags defines and parses command-line flags, returning them
// in a Config struct.
func parseFlags() *Config {
	cfg := &Config{}

	pflag.StringVarP(&cfg.InputPath, "input", "i", "", "Path to the input image.")
	pflag.StringVarP(&cfg.OutputDirectory, "output", "o", "./output", "Directory to save difference images.")
	pflag.IntVarP(&cfg.UnitSize, "unit-size", "s", 512, "The height and width of the square units to divide the image into.")
	pflag.Float64VarP(&cfg.Threshold, "threshold", "t", 3.0, "The CIEDE2000 Delta E threshold to consider units different.")
	pflag.IntVarP(&cfg.CPUCores, "cpu-cores", "c", runtime.NumCPU(), "Number of CPU cores to use for processing.")

	pflag.Parse()
	return cfg
}

// validateConfig checks if the provided configuration is valid.
func validateConfig(cfg *Config) error {
	if cfg.InputPath == "" {
		return fmt.Errorf("--input/-i flag is required")
	}
	if _, err := os.Stat(cfg.InputPath); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", cfg.InputPath)
	}
	if cfg.UnitSize <= 0 {
		return fmt.Errorf("--unit-size must be a positive integer")
	}
	if cfg.CPUCores <= 0 {
		return fmt.Errorf("--cpu-cores must be a positive integer")
	}
	return nil
}

// run is the main application logic.
func run(cfg *Config) error {
	log.Printf("Starting image comparison with %d CPU cores.", cfg.CPUCores)
	runtime.GOMAXPROCS(cfg.CPUCores)

	img, err := loadImage(cfg.InputPath)
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
	jobs := make(chan UnitPair, len(units))
	// A channel to receive pairs that are found to be different.
	results := make(chan UnitPair, len(units))

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
			jobs <- UnitPair{UnitA: units[i], UnitB: units[j]}
		}
	}
	close(jobs)

	wg.Wait()
	close(results)
	saveWg.Wait()

	log.Println("Processing complete.")
	return nil
}

// loadImage opens and decodes an image from the given file path.
func loadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("could not decode image: %w", err)
	}
	return img, nil
}

// splitImageIntoUnits divides a large image into a slice of smaller Unit structs.
func splitImageIntoUnits(img image.Image, unitSize int) []*Unit {
	bounds := img.Bounds()
	units := []*Unit{}

	for y := bounds.Min.Y; y < bounds.Max.Y; y += unitSize {
		for x := bounds.Min.X; x < bounds.Max.X; x += unitSize {
			rect := image.Rect(x, y, x+unitSize, y+unitSize).Intersect(bounds)
			if rect.Empty() {
				continue
			}

			subImg := img.(interface {
				SubImage(r image.Rectangle) image.Image
			}).SubImage(rect)

			units = append(units, &Unit{
				ID:  image.Point{X: x / unitSize, Y: y / unitSize},
				Img: subImg,
			})
		}
	}
	return units
}

// worker is a goroutine that receives UnitPairs, compares them, and sends
// differing pairs to the results channel.
func worker(wg *sync.WaitGroup, jobs <-chan UnitPair, results chan<- UnitPair, threshold float64) {
	defer wg.Done()
	for pair := range jobs {
		diff := compareUnits(pair.UnitA.Img, pair.UnitB.Img)
		if diff > threshold {
			results <- pair
		}
	}
}

// compareUnits calculates the average CIEDE2000 Delta E color difference
// between two images.
func compareUnits(imgA, imgB image.Image) float64 {
	boundsA := imgA.Bounds()
	boundsB := imgB.Bounds()

	// Ensure images are comparable in size.
	if boundsA.Dx() != boundsB.Dx() || boundsA.Dy() != boundsB.Dy() {
		return math.MaxFloat64 // Or handle error appropriately
	}

	var totalDifference float64
	pixelCount := float64(boundsA.Dx() * boundsA.Dy())

	for y := 0; y < boundsA.Dy(); y++ {
		for x := 0; x < boundsA.Dx(); x++ {
			colorA := toColorfulColor(imgA.At(boundsA.Min.X+x, boundsA.Min.Y+y))
			colorB := toColorfulColor(imgB.At(boundsB.Min.X+x, boundsB.Min.Y+y))
			totalDifference += colorA.DistanceCIEDE2000(colorB)
		}
	}

	if pixelCount == 0 {
		return 0
	}
	return totalDifference / pixelCount
}

// toColorfulColor converts a standard Go color.Color to a go-colorful.Color.
func toColorfulColor(c color.Color) colorful.Color {
	r, g, b, _ := c.RGBA()
	return colorful.Color{
		R: float64(r) / 65535.0,
		G: float64(g) / 65535.0,
		B: float64(b) / 65535.0,
	}
}

// saveDifferentPairs reads from the results channel and saves the image pairs
// to the specified output directory.
func saveDifferentPairs(results <-chan UnitPair, outputDir string) {
	for pair := range results {
		dirName := fmt.Sprintf("unit_%d_%d_vs_unit_%d_%d",
			pair.UnitA.ID.X, pair.UnitA.ID.Y,
			pair.UnitB.ID.X, pair.UnitB.ID.Y,
		)
		pairDir := filepath.Join(outputDir, dirName)

		if err := os.MkdirAll(pairDir, 0755); err != nil {
			log.Printf("Error creating directory %s: %v", pairDir, err)
			continue
		}

		saveUnit(pair.UnitA, pairDir)
		saveUnit(pair.UnitB, pairDir)
		log.Printf("Saved differing pair to %s", dirName)
	}
}

// saveUnit saves a single Unit's image to a PNG file.
func saveUnit(unit *Unit, dir string) {
	fileName := fmt.Sprintf("unit_%d_%d.png", unit.ID.X, unit.ID.Y)
	filePath := filepath.Join(dir, fileName)

	outFile, err := os.Create(filePath)
	if err != nil {
		log.Printf("Error creating file %s: %v", filePath, err)
		return
	}
	defer outFile.Close()

	if err := png.Encode(outFile, unit.Img); err != nil {
		log.Printf("Error encoding png %s: %v", filePath, err)
	}
}
