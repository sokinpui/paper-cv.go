package comparator

import (
	"fmt"
	"image"
	"strings"
)

// UnitComparator defines the interface for comparing two image units.
type UnitComparator interface {
	Compare(imgA, imgB image.Image) float64
}

// NewUnitComparator returns a comparator based on the specified device.
func NewUnitComparator(device string) (UnitComparator, error) {
	switch strings.ToLower(device) {
	case "cpu":
		return &CPUComparator{}, nil
	case "cuda":
		return nil, fmt.Errorf("CUDA support is not yet implemented")
	case "mps":
		return nil, fmt.Errorf("MPS support is not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported device: %s", device)
	}
}
