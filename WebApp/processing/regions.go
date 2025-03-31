package processing

import (
	"image"
	"image/color"
)

func BuildRegionMap(img image.Image, options RegionMapOptions, regionFilter func(Region) bool) *RegionMap {
	dx, dy := img.Bounds().Dx(), img.Bounds().Dy()
	regionMap := RegionMap{make([]Region, 0, 20), make([][]RegionId, dy), options}
	for i := range regionMap.pixels {
		regionMap.pixels[i] = make([]RegionId, dx)
	}

	bd := img.Bounds()

	for y := bd.Min.Y; y < bd.Max.Y; y++ {
		for x := bd.Min.X; x < bd.Max.X; x++ {
			pixel := Pixel{uint16(x), uint16(y)}
			if !regionMap.GetPixelHasRegion(pixel) {
				c := img.At(x, y)
				if c != Blank && (regionMap.options.AllowWhite || c != White) {
					regionMap.AddPixelToRegionMap(pixel, img)
				}
			}
		}
	}

	if regionFilter != nil {
		for i, region := range regionMap.regions {
			if region != nil && !regionFilter(region) {
				regionMap.regions[i] = nil
			}
		}
	}

	return &regionMap
}

type Pixel struct {
	X, Y uint16
}

type Vertex struct {
	X uint16 `json:"x"`
	Y uint16 `json:"y"`
}

type RegionId uint

func (r RegionId) GetRegionIndex() int {
	return int(r - 1)
}

type Region []Pixel

type RegionMap struct {
	regions []Region
	pixels  [][]RegionId
	options RegionMapOptions
}

type RegionMapOptions struct {
	NoColorSeparation, AllowWhite bool
}

func (rm *RegionMap) NewRegion(pixel Pixel) (region RegionId) {
	region = RegionId(len(rm.regions) + 1)
	rm.regions = append(rm.regions, Region{pixel})
	rm.pixels[pixel.Y][pixel.X] = region
	return
}

func (rm *RegionMap) AddPixelToRegion(pixel Pixel, region RegionId) {
	rm.regions[region.GetRegionIndex()] = append(rm.regions[region.GetRegionIndex()], pixel)
	rm.pixels[pixel.Y][pixel.X] = region
}

func (rm *RegionMap) AddPixelToRegionMap(pixel Pixel, img image.Image) {
	regionColor := img.At(int(pixel.X), int(pixel.Y))

	if !rm.GetPixelHasRegion(pixel) {
		region := rm.NewRegion(pixel)

		// iterative depth first traversal
		pixelsToVisit := make([]Pixel, 1, 8)
		pixelsToVisit[0] = pixel
		for len(pixelsToVisit) > 0 {
			cur := pixelsToVisit[len(pixelsToVisit)-1]
			pixelsToVisit = pixelsToVisit[:len(pixelsToVisit)-1]
			forNonDiagonalAdjacents(cur.X, cur.Y, len(rm.pixels[cur.Y]), len(rm.pixels), func(x, y uint16) {
				p := Pixel{x, y}
				if !rm.GetPixelHasRegion(p) && img.At(int(x), int(y)) == regionColor {
					rm.AddPixelToRegion(p, region)
					pixelsToVisit = append(pixelsToVisit, p)
				}
			})
		}
	}
}

func (rm *RegionMap) GetPixelHasRegion(pixel Pixel) (hasRegion bool) {
	return rm.pixels[pixel.Y][pixel.X] != 0
}

func (rm *RegionMap) GetRegionOfPixel(pixel Pixel) (regionId RegionId) {
	return rm.pixels[pixel.Y][pixel.X]
}

func (rm *RegionMap) GetRegion(region RegionId) Region {
	if rp := rm.regions[region.GetRegionIndex()]; rp != nil {
		return rp
	}
	return nil
}

func (rm *RegionMap) GetRegions() []Region {
	return rm.regions
}

func (re *Region) GetBounds() (regionBounds image.Rectangle) {
	regionBounds = image.Rectangle{Min: image.Pt(65535, 65535), Max: image.Pt(0, 0)}
	for _, pixel := range *re {
		if pixel.X < uint16(regionBounds.Min.X) {
			regionBounds.Min.X = int(pixel.X)
		}
		if pixel.Y < uint16(regionBounds.Min.Y) {
			regionBounds.Min.Y = int(pixel.Y)
		}
		if pixel.X+1 > uint16(regionBounds.Max.X) {
			regionBounds.Max.X = int(pixel.X) + 1
		}
		if pixel.Y+1 > uint16(regionBounds.Max.Y) {
			regionBounds.Max.Y = int(pixel.Y) + 1
		}
	}
	return
}

func ColorRegionEquivalence(a color.Color, b color.Color) bool {
	return a == b
}

func FindRegionPosition(region Region) (int, int) {
	corner := region[0]

	for i := 0; i < len(region); i++ {
		if region[i].X < corner.X {
			corner.X = region[i].X
		}
		if region[i].Y < corner.Y {
			corner.Y = region[i].Y
		}
	}

	return int(corner.X), int(corner.Y)
}

func GetColorOfRegion(region Region, img image.Image, checkAll bool) color.Color {
	if checkAll {
		colorCounts := make(map[color.Color]uint, 1)
		for _, v := range region {
			colorCounts[img.At(int(v.X), int(v.Y))]++
		}
		var mostCommonColor color.Color
		var mostCommonColorCount uint = 0
		for k, v := range colorCounts {
			if v > mostCommonColorCount {
				mostCommonColorCount = v
				mostCommonColor = k
			}
		}
		return mostCommonColor
	} else {
		regionColor := img.At(int(region[0].X), int(region[0].Y))
		return regionColor
	}
}
