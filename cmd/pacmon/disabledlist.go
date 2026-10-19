package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"larpix/pacmon/cmd/pacmon/monitors"
)

func channelsAboveThreshold(
	counts []monitors.DataStatusCounts,
	ioGroup uint8,
	duration float64,
	threshold float64,
) []monitors.ChannelKey {
	var channelsAbove []monitors.ChannelKey
	for channel, value := range counts {
		if float64(value.Total)/duration > threshold {
			channelsAbove = append(
				channelsAbove,
				monitors.ChannelKeyAt(ioGroup, channel),
			)
		}
	}
	return channelsAbove
}

func writeDisabledList(
	dlm *monitors.DisabledListMonitor,
	threshold float64,
	duration float64,
	ioGroup uint8,
) {

	hotChannels := channelsAboveThreshold(
		dlm.DataStatusCountsPerChannel[:],
		dlm.IoGroup,
		duration,
		threshold,
	)

	// Create a map to hold the values
	data := make(map[string][]int)
	for _, channel := range hotChannels {
		key := fmt.Sprintf(
			"%d-%d-%d",
			channel.IoGroup,
			monitors.IoChannelToTileId(int(channel.IoChannel)),
			channel.ChipID,
		)
		data[key] = append(data[key], int(channel.ChannelID))
	}

	// Create a JSON file
	now := time.Now()

	file, err := os.Create(
		fmt.Sprintf(
			"/data/disabledlist/iog_%d-"+
				"%d_%02d_%02d_%02d_%02d_%02d.json",
			ioGroup,
			now.Year(),
			now.Month(),
			now.Day(),
			now.Hour(),
			now.Minute(),
			now.Second(),
		),
	)
	if err != nil {
		fmt.Println("Error creating JSON file:", err)
		return
	}
	defer file.Close()

	// Marshal data to JSON
	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	// Write JSON data to file
	_, err = file.Write(jsonData)
	if err != nil {
		fmt.Println("Error writing JSON data to file:", err)
		return
	}

	fmt.Println("Disabled list written successfully on: ", now)
}
