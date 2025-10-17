package comparator

import (
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/lucasb-eyer/go-colorful"
)

// worker is a goroutine that receives unitPairs, compares them, and sends
// differing pairs to the results channel.
func worker(wg *sync.WaitGroup, jobs <-chan unitPair, results chan<- unitPair, threshold float64) {
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
