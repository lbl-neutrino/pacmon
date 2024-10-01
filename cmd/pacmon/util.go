package main

import (
	// "encoding/binary"
	"math"
	"sort"

	. "larpix/pacmon/pkg"
)

func Parity64(data Packet) byte {
	// x := binary.LittleEndian.Uint64(data)
	x := uint64(data[0])<<56 | uint64(data[1])<<48 | uint64(data[2])<<40 | uint64(data[3])<<32 |
		uint64(data[4])<<24 | uint64(data[5])<<16 | uint64(data[6])<<8 | uint64(data[7])
	x ^= x >> 32
	x ^= x >> 16
	x ^= x >> 8
	x ^= x >> 4
	x ^= x >> 2
	return byte(x) & 1
}

func UpdateMeanRMS(oldMean float64, oldRMS float64, oldNPackets uint32, newValue float64) (float64, float64) {
	// https://en.wikipedia.org/wiki/Algorithms_for_calculating_variance#Welford's_online_algorithm
	if oldNPackets == 0 {
		return newValue, 0.
	}

	newMean := oldMean + (newValue-oldMean)/(float64(oldNPackets)+1.)
	oldVariance := math.Pow(oldRMS, 2)
	newVariance := oldVariance + ((newValue-oldMean)*(newValue-newMean)-oldVariance)/(float64(oldNPackets)+1.)
	return newMean, math.Sqrt(newVariance)
}

// For sorting maps with arbitrary key types

func sortByDataRates(inputMap map[ChannelKey]DataStatusCounts, n int) ([]ChannelKey, []uint) {
	// Create a slice to hold the keys
	keys := make([]ChannelKey, 0, len(inputMap))

	// Iterate over the map and add keys to the slice
	for key := range inputMap {
		keys = append(keys, key)
	}

	// Sort the keys based on the corresponding values
	sort.Slice(keys, func(i, j int) bool {
		return inputMap[keys[i]].Total > inputMap[keys[j]].Total
	})

	// Extract top N keys and their corresponding values
	topNKeys := make([]ChannelKey, 0, n)
	topNValues := make([]uint, 0, n)
	for i := 0; i < n && i < len(keys); i++ {
		topNKeys = append(topNKeys, keys[i])
		topNValues = append(topNValues, inputMap[keys[i]].Total)
	}

	return topNKeys, topNValues
}

func sortByADC(inputMap map[ChannelKey]float64, n int) ([]ChannelKey, []float64) {
	// Create a slice to hold the keys
	keys := make([]ChannelKey, 0, len(inputMap))

	// Iterate over the map and add keys to the slice
	for key := range inputMap {
		keys = append(keys, key)
	}

	// Sort the keys based on the corresponding values
	sort.Slice(keys, func(i, j int) bool {
		return inputMap[keys[i]] > inputMap[keys[j]]
	})

	// Extract top N keys and their corresponding values
	topNKeys := make([]ChannelKey, 0, n)
	topNValues := make([]float64, 0, n)
	for i := 0; i < n && i < len(keys); i++ {
		topNKeys = append(topNKeys, keys[i])
		topNValues = append(topNValues, inputMap[keys[i]])
	}

	return topNKeys, topNValues
}
