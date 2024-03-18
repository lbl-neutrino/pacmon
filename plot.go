package main

import (
	"fmt"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/palette"
	"gonum.org/v1/plot/palette/moreland"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/font"
	// "gonum.org/v1/plot/vg/vgimg"
	// "gonum.org/v1/gonum/mat"
	"math"
	"log"
	"time"
	// "os"
)

type plottable struct {
	grid [][]float64
	N int
	M int
	resolution float64
	pos [][][]float64
}


// func (p plottable) Dims() (c, r int) {
// 	return p.N, p.M
// }

// func (p plottable) X(c int) float64 {
// 	return p.minX + float64(c)*p.resolution
// }
// func (p plottable) Y(r int) float64 {
// 	return p.minY + float64(r)*p.resolution
// }
// func (p plottable) Z(c, r int) float64 {
// 	return p.grid[c][r]
// }

// func PlotTest() {

// 	p := plot.New()
	
// 	// Define some XYZ data

// 	length := 10
// 	xyzs := make(plotter.XYZs, length)
// 	for i := 0; i < length; i++ {
// 		xyzs[i].X = rand.Float64()
// 		xyzs[i].Y = xyzs[i].X/2 + 0.2 * rand.Float64()
// 		xyzs[i].Z = rand.Float64()
// 	}

// 	// Get the min/max of the z-axis 
// 	minZ, maxZ := math.Inf(1), math.Inf(-1)
// 	for _, xyz := range xyzs {
// 		if xyz.Z > maxZ {
// 			maxZ = xyz.Z
// 		}
// 		if xyz.Z < minZ {
// 			minZ = xyz.Z
// 		}
// 	}

// 	// Initialize a color map
// 	colors := moreland.Kindlmann() 
// 	colors.SetMax(maxZ)
// 	colors.SetMin(minZ)

// 	sc, err := plotter.NewScatter(xyzs)
// 	if err != nil {
// 		panic(err)
// 	}
	
// 	// Setup the style options
// 	sc.GlyphStyleFunc = func(i int) draw.GlyphStyle {
// 		_, _, z := xyzs.XYZ(i)
// 		d := (z - minZ) / (maxZ - minZ)
// 		rng := maxZ - minZ
// 		k := d*rng + minZ
// 		c, err := colors.At(k)
// 		if err != nil {
// 			panic(err)
// 		}
// 		return draw.GlyphStyle{Color: c, Radius: 0.5*vg.Centimeter, Shape: draw.BoxGlyph{}}
// 	}

// 	// Add scatter data
// 	p.Add(sc)

// 	// Make colorbar by hand
// 	// thumbs := plotter.PaletteThumbnailers(colors.Palette(50))
// 	// for i := len(thumbs) - 1; i >= 0; i-- {
// 	// 	t := thumbs[i]
// 	// 	if i != 0 && i != len(thumbs)-1 {
// 	// 		p.Legend.Add("", t)
// 	// 		continue
// 	// 	}
// 	// 	var val int
// 	// 	switch i {
// 	// 	case 0:
// 	// 		val = int(minZ)
// 	// 	case len(thumbs) - 1:
// 	// 		val = int(maxZ)
// 	// 	}
// 	// 	p.Legend.Add(fmt.Sprintf("%d", val), t)
// 	// }

// 	// p.Legend.ThumbnailWidth = 0.5 * vg.Centimeter
// 	// const legendWidth = vg.Centimeter

// 	// p.Legend.XOffs = vg.Centimeter

// 	// img := vgimg.New(30*vg.Centimeter, 30*vg.Centimeter)
// 	// dc := draw.New(img)
// 	// dc = draw.Crop(dc, 0, -2*legendWidth, 0, -vg.Centimeter) // Make space for the legend.
// 	// p.Draw(dc)

// 	// p.X.Padding = 0*vg.Centimeter

// 	// l := &plotter.ColorBar{ColorMap: colors}
// 	// p.Add(l)

// 	if err := p.Save(30*vg.Centimeter, 30*vg.Centimeter, "test.png"); err != nil {
// 		log.Panic(err)
// 	}
// 	// c := vgimg.PngCanvas{Canvas: img}
// 	// // p.Draw(draw.New(c))

// 	// f, err := os.Create("test.png")
// 	// if err != nil {
// 	// 	log.Fatalf("could not create output image file: %+v", err)
// 	// }
// 	// defer f.Close()

// 	// _, err = c.WriteTo(f)
// 	// if err != nil {
// 	// 	log.Fatalf("could not encode image to PNG: %+v", err)
// 	// }

// 	// err = f.Close()
// 	// if err != nil {
// 	// 	log.Fatalf("could not close output image file: %+v", err)
// 	// }

// }

func (m1min *Monitor1min) PlotMean(geometry Geometry) {
	norm := 50.

	fmt.Println(time.Now(), ": start plotting")

	p := plot.New()
	length := len(m1min.ADCMeanPerChannel)

	xyzs := make(plotter.XYZs, length)
	
	i := 0

	for channelKey, adc := range m1min.ADCMeanPerChannel {

		// convert from ChannelKey to ChannelTile
		var channelTile ChannelTile
		channelTile.IoGroup = channelKey.IoGroup
		channelTile.TileID = (uint8(channelKey.IoChannel) - 1)/4 + 1 + 8 * (1 - (channelKey.IoGroup % 2))
		channelTile.ChipID = channelKey.ChipID
		channelTile.ChannelID = channelKey.ChannelID
		
		// Get XY positions
		xy, ok := geometry.ChannelToXY[channelTile]
		if !ok {
			fmt.Println(channelTile, " not found in the geometry")
			continue
		}

		xyzs[i].X = xy.X
		xyzs[i].Y = xy.Y
		xyzs[i].Z = adc/norm
		if xyzs[i].Z > 1 {
			xyzs[i].Z = 1.
		}
		i++
	}

	// Get the min/max of the z-axis 
	minZ, maxZ := math.Inf(1), math.Inf(-1)
	minX, maxX := math.Inf(1), math.Inf(-1)
	minY, maxY := math.Inf(1), math.Inf(-1)
	for _, xy := range geometry.ChannelToXY {

		if xy.X > maxX {
			maxX = xy.X
		}
		if xy.X < minX {
			minX = xy.X
		}

		if xy.Y > maxY {
			maxY = xy.Y
		}
		if xy.Y < minY {
			minY = xy.Y
		}

	}
	maxZ = 1
	minZ = 0
	// Initialize a color map
	colors := palette.Reverse(moreland.BlackBody())
	colors.SetMax(maxZ)
	colors.SetMin(minZ)

	sc, err := plotter.NewScatter(xyzs)
	if err != nil {
		panic(err)
	}
	
	// Setup the style options
	sc.GlyphStyleFunc = func(i int) draw.GlyphStyle {
		_, _, z := xyzs.XYZ(i)
		d := (z - minZ) / (maxZ - minZ)
		rng := maxZ - minZ
		k := d*rng + minZ
		c, err := colors.At(k)
		if err != nil {
			panic(err)
		}
		return draw.GlyphStyle{Color: c, 
			Radius: font.Length(geometry.Pitch*vg.Millimeter.Points()), 
			Shape: draw.BoxGlyph{},
		}
	}

	p.Add(sc)

	fmt.Println(time.Now(), ": saving plot")
	if err := p.Save(650*vg.Millimeter, 650*vg.Millimeter, "test.png"); err != nil {
		log.Panic(err)
	}


}
