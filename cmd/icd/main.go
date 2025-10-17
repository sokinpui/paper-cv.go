package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/sokinpui/paper-cv/internal/comparator"
	"github.com/sokinpui/paper-cv/internal/logger"
	"github.com/spf13/pflag"
)

func main() {
	logFile, err := logger.Init("comparator.log")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	cfg := parseFlags()
	if err = validateConfig(cfg); err != nil {
		log.Printf("Configuration error: %v", err)
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	if err = comparator.Run(cfg); err != nil {
		log.Printf("Application error: %v", err)
		fmt.Fprintf(os.Stderr, "Application error: %v\n", err)
		os.Exit(1)
	}
}

// parseFlags defines and parses command-line flags, returning them
// in a Config struct.
func parseFlags() *comparator.Config {
	cfg := &comparator.Config{}

	pflag.StringVarP(&cfg.InputPath, "input", "i", "", "Path to the input image.")
	pflag.StringVarP(&cfg.OutputDirectory, "output", "o", "./output", "Directory to save difference images.")
	pflag.IntVarP(&cfg.UnitSize, "unit-size", "s", 512, "The height and width of the square units to divide the image into.")
	pflag.Float64VarP(&cfg.Threshold, "threshold", "t", 3.0, "The CIEDE2000 Delta E threshold to consider units different.")
	pflag.IntVarP(&cfg.CPUCores, "cpu-cores", "c", runtime.NumCPU(), "Number of CPU cores to use for processing.")
	pflag.StringVar(&cfg.ImageType, "type", "", "Type of the input image (e.g., jpeg, png, bmp). If not specified, it will be inferred.")
	pflag.StringVarP(&cfg.Device, "device", "d", "cpu", "Device to use for processing (cpu, cuda, mps).")

	pflag.Parse()
	return cfg
}

// validateConfig checks if the provided configuration is valid.
func validateConfig(cfg *comparator.Config) error {
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
	if cfg.ImageType != "" {
		switch cfg.ImageType {
		case "jpeg", "jpg", "png", "bmp":
		default:
			return fmt.Errorf("unsupported image type: %s", cfg.ImageType)
		}
	}
	switch strings.ToLower(cfg.Device) {
	case "cpu", "cuda", "mps":
	default:
		return fmt.Errorf("unsupported device: %s. Supported devices are cpu, cuda, mps", cfg.Device)
	}
	return nil
}
