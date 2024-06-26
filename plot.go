package main

import (
	"fmt"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/font"
	"gonum.org/v1/plot/palette"
	"gonum.org/v1/plot/palette/moreland"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"

	// "gonum.org/v1/plot/vg/vgimg"
	// "gonum.org/v1/gonum/mat"
	"log"
	"math"
	"time"
	"os"
)

func (mPlots *MonitorPlots) PlotMetrics(geometry Geometry, ioGroup uint8, norms Norms, timeDiff float64) {

	length := len(mPlots.ADCMeanPerChannel)
	if length == 0 {
		return
	}

	normMean := norms.Mean
	normRMS := norms.RMS
	normRate := norms.Rate

	fmt.Println(time.Now(), ": start plotting for io_group = ", ioGroup)

	pMean := plot.New()
	pMean.X.Tick.Label.Font.Size = 30
	pMean.Y.Tick.Label.Font.Size = 30

	pRMS := plot.New()
	pRMS.X.Tick.Label.Font.Size = 30
	pRMS.Y.Tick.Label.Font.Size = 30

	pRate := plot.New()
	pRate.X.Tick.Label.Font.Size = 30
	pRate.Y.Tick.Label.Font.Size = 30

	xyzsMean := make(plotter.XYZs, length)
	xyzsRMS := make(plotter.XYZs, length)
	xyzsRate := make(plotter.XYZs, length)

	i := 0

	for channelKey, adc := range mPlots.ADCMeanPerChannel {

		// convert from ChannelKey to ChannelTile
		var channelTile ChannelTile
		channelTile.IoGroup = 2 - (channelKey.IoGroup % 2)
		channelTile.TileID = (uint8(channelKey.IoChannel)-1)/4 + 1 + 8*(1-(channelKey.IoGroup%2))
		channelTile.ChipID = channelKey.ChipID
		channelTile.ChannelID = channelKey.ChannelID

		// Get XY positions
		xy, ok := geometry.ChannelToXY[channelTile]
		if !ok {
			fmt.Println(channelTile, " not found in the geometry")
			continue
		}

		// Mean
		xyzsMean[i].X = xy.X
		xyzsMean[i].Y = xy.Y
		xyzsMean[i].Z = adc / normMean
		if xyzsMean[i].Z > 1 {
			xyzsMean[i].Z = 1.
		}

		// RMS
		xyzsRMS[i].X = xy.X
		xyzsRMS[i].Y = xy.Y
		xyzsRMS[i].Z = mPlots.ADCRMSPerChannel[channelKey] / normRMS
		if xyzsRMS[i].Z > 1 {
			xyzsRMS[i].Z = 1.
		}

		// Rate
		xyzsRate[i].X = xy.X
		xyzsRate[i].Y = xy.Y
		xyzsRate[i].Z = (float64(mPlots.NPacketsPerChannel[channelKey]) / timeDiff) / normRate
		if xyzsRate[i].Z > 1 {
			xyzsRate[i].Z = 1.
		}

		i++
	}

	// Get the min/max of the z-axis
	// minZ, maxZ := math.Inf(1), math.Inf(-1)
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

	maxZ := 1.
	minZ := 0.
	// Initialize a color map

	colors := palette.Reverse(moreland.BlackBody())
	colors.SetMax(maxZ)
	colors.SetMin(minZ)

	pMean.X.Min = minX
	pMean.X.Max = maxX
	pMean.Y.Min = minY
	pMean.Y.Max = maxY

	pRMS.X.Min = minX
	pRMS.X.Max = maxX
	pRMS.Y.Min = minY
	pRMS.Y.Max = maxY

	pRate.X.Min = minX
	pRate.X.Max = maxX
	pRate.Y.Min = minY
	pRate.Y.Max = maxY

	scMean, err := plotter.NewScatter(xyzsMean)
	if err != nil {
		panic(err)
	}

	scRMS, err := plotter.NewScatter(xyzsRMS)
	if err != nil {
		panic(err)
	}

	scRate, err := plotter.NewScatter(xyzsRate)
	if err != nil {
		panic(err)
	}

	// Setup the style options
	scMean.GlyphStyleFunc = func(i int) draw.GlyphStyle {
		_, _, z := xyzsMean.XYZ(i)
		d := (z - minZ) / (maxZ - minZ)
		rng := maxZ - minZ
		k := d*rng + minZ
		c, err := colors.At(k)
		if err != nil {
			panic(err)
		}
		return draw.GlyphStyle{Color: c,
			Radius: font.Length(geometry.Pitch * vg.Millimeter.Points() / 2.),
			Shape:  draw.BoxGlyph{},
		}
	}
	scRMS.GlyphStyleFunc = func(i int) draw.GlyphStyle {
		_, _, z := xyzsRMS.XYZ(i)
		d := (z - minZ) / (maxZ - minZ)
		rng := maxZ - minZ
		k := d*rng + minZ
		c, err := colors.At(k)
		if err != nil {
			panic(err)
		}
		return draw.GlyphStyle{Color: c,
			Radius: font.Length(geometry.Pitch * vg.Millimeter.Points() / 2.),
			Shape:  draw.BoxGlyph{},
		}
	}
	scRate.GlyphStyleFunc = func(i int) draw.GlyphStyle {
		_, _, z := xyzsRate.XYZ(i)
		d := (z - minZ) / (maxZ - minZ)
		rng := maxZ - minZ
		k := d*rng + minZ
		c, err := colors.At(k)
		if err != nil {
			panic(err)
		}
		return draw.GlyphStyle{Color: c,
			Radius: font.Length(geometry.Pitch * vg.Millimeter.Points() / 2.),
			Shape:  draw.BoxGlyph{},
		}
	}

	pMean.Add(scMean)
	pRMS.Add(scRMS)
	pRate.Add(scRate)

	now := time.Now()
	pMean.Title.Text = fmt.Sprintf("ADC Mean - io_group = %d\n%d-%02d-%02d %02d:%02d:%02d", ioGroup, now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	pRMS.Title.Text = fmt.Sprintf("ADC RMS - io_group = %d\n%d-%02d-%02d %02d:%02d:%02d", ioGroup, now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	pRate.Title.Text = fmt.Sprintf("Rate - io_group = %d\n%d-%02d-%02d %02d:%02d:%02d", ioGroup, now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())

	pMean.Title.TextStyle.Font.Size = 120
	pRMS.Title.TextStyle.Font.Size = 120
	pRate.Title.TextStyle.Font.Size = 120

	fmt.Println(now, ": saving plot for io_group = ", ioGroup)
	// Save for history
	if err := pMean.Save(font.Length((maxX-minX)*vg.Millimeter.Points()+30.), font.Length((maxY-minY)*vg.Millimeter.Points()+150.), fmt.Sprintf("/data/plots/mean/iog_%d_mean_%d_%02d_%02d_%02d_%02d_%02d.png", ioGroup, now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())); err != nil {
		log.Panic(err)
	}
	if err := pRMS.Save(font.Length((maxX-minX)*vg.Millimeter.Points()+30.), font.Length((maxY-minY)*vg.Millimeter.Points()+150.), fmt.Sprintf("/data/plots/rms/iog_%d_rms_%d_%02d_%02d_%02d_%02d_%02d.png", ioGroup, now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())); err != nil {
		log.Panic(err)
	}
	if err := pRate.Save(font.Length((maxX-minX)*vg.Millimeter.Points()+30.), font.Length((maxY-minY)*vg.Millimeter.Points()+150.), fmt.Sprintf("/data/plots/rate/iog_%d_rate_%d_%02d_%02d_%02d_%02d_%02d.png", ioGroup, now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())); err != nil {
		log.Panic(err)
	}
	// Save for instant updates
	if err := pMean.Save(font.Length((maxX-minX)*vg.Millimeter.Points()+30.), font.Length((maxY-minY)*vg.Millimeter.Points()+150.), fmt.Sprintf("/data/plots/RealTime/iog_%d_mean.png", ioGroup)); err != nil {
		log.Panic(err)
	}
	s := fmt.Sprintf("%d-%02d-%02d\n%02d:%02d:%02d", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	f, err := os.Create(fmt.Sprintf("/data/plots/RealTime/iog_%d_last_update.txt", ioGroup))
	if err != nil {
		panic(err)
	}
	_, err = f.WriteString(s)
	if err != nil {
		fmt.Println(err)
	        f.Close()
	}
	if err := pRMS.Save(font.Length((maxX-minX)*vg.Millimeter.Points()+30.), font.Length((maxY-minY)*vg.Millimeter.Points()+150.), fmt.Sprintf("/data/plots/RealTime/iog_%d_rms.png", ioGroup)); err != nil {
		log.Panic(err)
	}
	if err := pRate.Save(font.Length((maxX-minX)*vg.Millimeter.Points()+30.), font.Length((maxY-minY)*vg.Millimeter.Points()+150.), fmt.Sprintf("/data/plots/RealTime/iog_%d_rate.png", ioGroup)); err != nil {
		log.Panic(err)
	}

}
