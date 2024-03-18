package main

import (
	"fmt"
    "encoding/json"
	"io/ioutil"
	"strings"
	"strconv"
)

type XY struct {
	X float64
	Y float64
}

type ChannelTile struct {
	IoGroup uint8
	TileID uint8
	ChipID uint8
	ChannelID uint8
}
type Geometry struct {
	Pitch float64
	ChannelToXY map[ChannelTile]XY
}

type GeoConfig struct {
	Pitch float64 `json:"pixel_pitch"`
	Geometry map[string][2]float64 `json:"geometry"`
}

func ReadGeometryFile(path string) GeoConfig {
    content, err := ioutil.ReadFile(path)
	if err == nil {
        fmt.Println("Reading geometry config from JSON file...")

		var config GeoConfig

		err = json.Unmarshal([]byte(content), &config)
		if err != nil {
			panic(err)
		}
		// fmt.Println(config.Geometry["1-7-11-63"])
		return config
	} else {
		panic("File not found.")
	}

}

func LoadGeometry(path string) Geometry {
	var geo Geometry
	config := ReadGeometryFile(path)

	geo.Pitch = config.Pitch
	geo.ChannelToXY = make(map[ChannelTile]XY)

	for str, pos := range config.Geometry {
	
		var channel ChannelTile
		s := strings.Split(str, "-")

		ioGroup, err := strconv.Atoi(s[0])
		if err != nil {
			panic(err)
		}
		channel.IoGroup = uint8(ioGroup)

		tileID, err := strconv.Atoi(s[1])
		if err != nil {
			panic(err)
		}
		channel.TileID = uint8(tileID)
		
		chipID, err := strconv.Atoi(s[2])
		if err != nil {
			panic(err)
		}
		channel.ChipID = uint8(chipID)
		
		channelID, err := strconv.Atoi(s[3])
		if err != nil {
			panic(err)
		}
		channel.ChannelID = uint8(channelID)
		
		var xy XY
		xy.X = pos[0]
		xy.Y = pos[1]

		geo.ChannelToXY[channel] = xy
	}

	var channel ChannelTile
	channel.IoGroup = 1
	channel.TileID = 7
	channel.ChipID = 11
	channel.ChannelID = 63

	// fmt.Println(config.Geometry["1-7-11-63"])
	// fmt.Println(geo.ChannelToXY[channel])
	

	return geo
}
