package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type XY struct {
	X float64
	Y float64
}

type ChannelTile struct {
	IoGroup   uint8
	TileID    uint8
	ChipID    uint8
	ChannelID uint8
}

type Geometry struct {
	Pitch       float64
	ChannelToXY map[ChannelTile]XY
	IoGroupIoChannelToTile map[IoGroupIoChannel]int
}

type GeoConfig struct {
	Pitch    float64               `json:"pixel_pitch"`
	Geometry map[string][2]float64 `json:"geometry"`
	// Keys are tile IDs (JSON object keys are decoded from strings to ints)
	TileToIoChannels map[int][]int `json:"tile_to_io_channels"`
	// Keys are "<io_group>-<io_channel>" strings in the JSON file
	IoGroupIoChannelToTile map[IoGroupIoChannel]int `json:"io_group_io_channel_to_tile"`
}

type IoGroupIoChannel struct {
	IoGroup   int
	IoChannel int
}

// Lets encoding/json decode "<io_group>-<io_channel>" object keys
func (k *IoGroupIoChannel) UnmarshalText(text []byte) error {
	_, err := fmt.Sscanf(string(text), "%d-%d", &k.IoGroup, &k.IoChannel)
	return err
}

func ReadGeometryFile(path string) GeoConfig {
	content, err := os.ReadFile(path)
	if err == nil {
		fmt.Println("Reading geometry config from JSON file: ", path)

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
	geo.IoGroupIoChannelToTile = config.IoGroupIoChannelToTile
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

	// Quick check

	// var channel ChannelTile
	// channel.IoGroup = 1
	// channel.TileID = 7
	// channel.ChipID = 11
	// channel.ChannelID = 63

	// fmt.Println(config.Geometry["1-7-11-63"])
	// fmt.Println(geo.ChannelToXY[channel])

	return geo
}
