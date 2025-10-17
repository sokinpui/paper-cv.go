package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/sokinpui/paper-cv/internal/comparator"
	"github.com/spf13/pflag"
)

func main() {
	cfg := parseFlags()
	if err := validateConfig(cfg); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	if err := comparator.Run(cfg); err != nil {
		log.Fatalf("Application error: %v", err)
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
	return nil
}
