package processing

import (
	"image"
	"image/png"
	"os"
	"testing"
)

func loadImage(path string) image.Image {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	i, err := png.Decode(f)
	if err != nil {
		panic(err)
	}

	return i
}

func BenchmarkBuildRegionMap(b *testing.B) {
	type args struct {
		img          image.Image
		options      RegionMapOptions
		regionFilter func(Region) bool
	}
	benchmarks := []struct {
		name string
		args args
	}{
		{
			name: "lub",
			args: args{
				img: loadImage("./build_region_map_test_images/lub.png"),
			},
		},
		{
			name: "whiteboardshapes",
			args: args{
				img: loadImage("./build_region_map_test_images/whiteboardshapes.png"),
			},
		},
		{
			name: "allwhite",
			args: args{
				img: loadImage("./build_region_map_test_images/allwhite.png"),
			},
		},
		{
			name: "allblack",
			args: args{
				img: loadImage("./build_region_map_test_images/allblack.png"),
			},
		},
		{
			name: "allcolors",
			args: args{
				img: loadImage("./build_region_map_test_images/allcolors.png"),
			},
		},
	}
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			img := SimplifyImage(bm.args.img, bm.args.options)
			for b.Loop() {
				BuildRegionMap(img, bm.args.options, bm.args.regionFilter)
			}
		})
	}
}
