package comparator

import (
	"image"
)

// unit represents a single square subdivision of the source image.
// It holds the image data and its coordinate position in the grid.
type unit struct {
	ID  image.Point
	Img image.Image
}

// unitPair represents a pair of units that need to be compared.
type unitPair struct {
	UnitA *unit
	UnitB *unit
}

// splitImageIntoUnits divides a large image into a slice of smaller unit structs.
func splitImageIntoUnits(img image.Image, unitSize int) []*unit {
	bounds := img.Bounds()
	units := []*unit{}

	for y := bounds.Min.Y; y < bounds.Max.Y; y += unitSize {
		for x := bounds.Min.X; x < bounds.Max.X; x += unitSize {
			rect := image.Rect(x, y, x+unitSize, y+unitSize).Intersect(bounds)
			if rect.Empty() {
				continue
			}

			subImg := img.(interface {
				SubImage(r image.Rectangle) image.Image
			}).SubImage(rect)

			units = append(units, &unit{
				ID:  image.Point{X: x / unitSize, Y: y / unitSize},
				Img: subImg,
			})
		}
	}
	return units
}
