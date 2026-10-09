package damage

import (
	"image"
	"testing"
)

func TestMapMatchesNearestNeighborSamples(t *testing.T) {
	t.Parallel()
	for sourceWidth := 1; sourceWidth <= 17; sourceWidth++ {
		for width := 1; width <= 19; width++ {
			source := image.Rect(-7, -3, sourceWidth-7, 8)
			output := image.Rect(4, -11, width+4, 2)
			for left := -1; left <= sourceWidth; left++ {
				for right := left; right <= sourceWidth+1; right++ {
					region := image.Rect(left-7, -1, right-7, 5)
					mapped := Map(region, source, output)
					for y := output.Min.Y; y < output.Max.Y; y++ {
						for x := output.Min.X; x < output.Max.X; x++ {
							sample := image.Pt(source.Min.X+(x-output.Min.X)*source.Dx()/output.Dx(),
								source.Min.Y+(y-output.Min.Y)*source.Dy()/output.Dy())
							if got, want := image.Pt(x, y).In(mapped), sample.In(region); got != want {
								t.Fatalf("source %v output %v region %v mapped %v pixel (%d,%d): covered %t, want %t",
									source, output, region, mapped, x, y, got, want)
							}
						}
					}
				}
			}
		}
	}
}

func TestAlignClipsAndPreservesEmptyDamage(t *testing.T) {
	t.Parallel()
	bounds := image.Rect(3, -7, 20, 16)
	for _, test := range []struct {
		region, want image.Rectangle
	}{
		{image.Rect(4, -6, 5, -5), image.Rect(3, -7, 5, -5)},
		{image.Rect(19, 15, 30, 40), image.Rect(19, 15, 20, 16)},
		{image.Rect(40, 40, 50, 50), image.Rectangle{}},
		{image.Rect(4, 4, 4, 9), image.Rectangle{}},
	} {
		if got := Align(test.region, 2, bounds); got != test.want {
			t.Fatalf("Align(%v) = %v, want %v", test.region, got, test.want)
		}
	}
}
