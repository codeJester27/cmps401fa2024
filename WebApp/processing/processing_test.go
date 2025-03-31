package processing

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"regexp"
	"slices"
	"testing"
)

var Cyan color.NRGBA = color.NRGBA{uint8(0), uint8(255), uint8(255), uint8(255)}
var Magenta color.NRGBA = color.NRGBA{uint8(255), uint8(0), uint8(255), uint8(255)}
var Yellow color.NRGBA = color.NRGBA{uint8(255), uint8(255), uint8(0), uint8(255)}

func generateRegion(img image.Image) *Region {
	region := make(Region, 0, img.Bounds().Dx()*img.Bounds().Dy())
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if c := color.NRGBAModel.Convert(img.At(x, y)); c != White {
				region = append(region, Pixel{uint16(x), uint16(y)})
			}
		}
	}
	return &region
}

func generateShape(img image.Image) *[]Vertex {
	shape := make([]Vertex, 0, img.Bounds().Dx()*img.Bounds().Dy())
	var x, y int

Y:
	for y = img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x = img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if img.At(x, y) != Blank {
				break Y
			}
		}
	}

	for {
		shape = append(shape, Vertex{uint16(x), uint16(y)})
		c := color.NRGBAModel.Convert(img.At(x, y))
		switch c {
		case Red:
			x--
			y--
		case Green:
			y--
		case Blue:
			x++
			y--
		case Black:
			x--
		case White:
			x++
		case Cyan:
			x--
			y++
		case Magenta:
			y++
		case Yellow:
			x++
			y++
		default:
			panic("your vertex map sucks")
		}

		if shape[0].X == uint16(x) && shape[0].Y == uint16(y) {
			break
		}
	}

	return &shape
}

func generateTestResultImage(name string, shape *[]Vertex) error {
	file, err := os.Create(fmt.Sprintf("./create_shape_test_images/test_out_%s.gif", name))
	if err != nil {
		return err
	}

	bounds := image.Rectangle{Min: image.Pt(65535, 65535), Max: image.Pt(0, 0)}
	for _, pixel := range *shape {
		if pixel.X+1 < uint16(bounds.Min.X) {
			bounds.Min.X = int(pixel.X)
		}
		if pixel.Y < uint16(bounds.Min.Y) {
			bounds.Min.Y = int(pixel.Y)
		}
		if pixel.X+1 > uint16(bounds.Max.X) {
			bounds.Max.X = int(pixel.X) + 1
		}
		if pixel.Y+1 > uint16(bounds.Max.Y) {
			bounds.Max.Y = int(pixel.Y) + 1
		}
	}
	plt := color.Palette{White, Black, Red}
	baseImg := image.NewPaletted(bounds, plt)
	images := make([]*image.Paletted, len(*shape))
	delays := make([]int, len(*shape))
	for _, v := range *shape {
		baseImg.Set(int(v.X), int(v.Y), Black)
	}

	for i, v := range *shape {
		img := *baseImg
		img.Pix = make([]uint8, len(baseImg.Pix))
		copy(img.Pix, baseImg.Pix)
		img.Set(int(v.X), int(v.Y), Red)
		images[i] = &img
		delays[i] = 10
	}

	g := &gif.GIF{
		Image: images,
		Delay: delays,
		Config: image.Config{
			ColorModel: plt,
			Width:      bounds.Max.X,
			Height:     bounds.Max.Y,
		},
	}

	err = gif.EncodeAll(file, g)
	if err != nil {
		return err
	}

	return file.Close()
}

var vertexMapNameRegex = regexp.MustCompile(`^test_(\w+)_vertexmap`)

func TestRegion_CreateShape(t *testing.T) {
	tests := make([]struct {
		name      string
		region    *Region
		wantShape *[]Vertex
	}, 0)

	testDir, err := os.ReadDir("./create_shape_test_images")

	if err != nil {
		panic(err)
	}

	for _, v := range testDir {
		if !v.IsDir() {
			matches := vertexMapNameRegex.FindStringSubmatch(v.Name())
			if matches != nil {
				tests = append(tests, struct {
					name      string
					region    *Region
					wantShape *[]Vertex
				}{name: matches[1]})
			}
		}
	}

	// use supplied test images
	for i, tt := range tests {
		regionf, err := os.Open(fmt.Sprintf("./create_shape_test_images/test_%s.png", tt.name))
		if err != nil {
			panic(err)
		}

		regionimg, err := png.Decode(regionf)
		if err != nil {
			panic(err)
		}

		tests[i].region = generateRegion(regionimg)

		vertexf, err := os.Open(fmt.Sprintf("./create_shape_test_images/test_%s_vertexmap.png", tt.name))
		if err != nil {
			panic(err)
		}

		verteximg, err := png.Decode(vertexf)
		if err != nil {
			panic(err)
		}

		tests[i].wantShape = generateShape(verteximg)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotShapeP, err := tt.region.CreateShape()
			if err != nil {
				t.Fatalf("Error: %s", err)
			}
			gotShape := gotShapeP
			wantShape := *tt.wantShape

			fail := func() {
				if generateTestResultImage(tt.name, &gotShape) == nil {
					t.Logf("Generated test result image for %s", tt.name)
				}
				t.Fatalf("Region.CreateShape() = %v\n want %v", gotShape, wantShape)
			}

			firstIndex := slices.Index(wantShape, gotShape[0])
			if firstIndex == -1 || len(wantShape) != len(gotShape) {
				fail()
			}

			length := len(gotShape)
			if gotShape[1] != wantShape[(firstIndex+1)%length] {
				// second elements don't match, let's try reverse?
				slices.Reverse(wantShape)
				firstIndex = (length - 1) - firstIndex
				if gotShape[1] != wantShape[(firstIndex+1)%length] {
					fail()
				}
			}
			for i := 2; i < length; i++ {
				if gotShape[i] != wantShape[(firstIndex+i)%length] {
					fail()
				}
			}
			if generateTestResultImage(tt.name, &gotShape) == nil {
				t.Logf("Generated test result image for %s", tt.name)
			}
		})
	}
}
