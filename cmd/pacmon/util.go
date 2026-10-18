package main

import (
	// "encoding/binary"
	"cmp"
	"math"
	"slices"

	. "larpix/pacmon/pkg"
)

func Parity64(data Packet) byte {
	// x := binary.LittleEndian.Uint64(data)
	x := uint64(
		data[0],
	)<<56 | uint64(
		data[1],
	)<<48 | uint64(
		data[2],
	)<<40 | uint64(
		data[3],
	)<<32 |
		uint64(
			data[4],
		)<<24 | uint64(
		data[5],
	)<<16 | uint64(
		data[6],
	)<<8 | uint64(
		data[7],
	)
	x ^= x >> 32
	x ^= x >> 16
	x ^= x >> 8
	x ^= x >> 4
	x ^= x >> 2
	return byte(x) & 1
}

func UpdateMeanRMS(
	oldMean float64,
	oldRMS float64,
	oldNPackets uint32,
	newValue float64,
) (float64, float64) {
	// Welford's online algorithm, see
	// https://en.wikipedia.org/wiki/Algorithms_for_calculating_variance
	if oldNPackets == 0 {
		return newValue, 0.
	}

	n := float64(oldNPackets) + 1.
	newMean := oldMean + (newValue-oldMean)/n
	oldVariance := math.Pow(oldRMS, 2)
	newVariance := oldVariance +
		((newValue-oldMean)*(newValue-newMean)-oldVariance)/n
	return newMean, math.Sqrt(newVariance)
}

// sortByValue returns the keys and values of the n largest (or smallest, if
// ascending) entries of values, considering only indices i for which
// present(i) is true. keyAt converts an index back into a key.
func sortByValue[K any, V cmp.Ordered](
	values []V,
	present func(i int) bool,
	n int,
	ascending bool,
	ioGroup uint8,
	keyAt func(ioGroup uint8, i int) K,
) ([]K, []V) {
	indices := make([]int, 0, len(values))
	for i := range values {
		if present(i) {
			indices = append(indices, i)
		}
	}

	slices.SortFunc(indices, func(a, b int) int {
		if ascending {
			return cmp.Compare(values[a], values[b])
		}
		return cmp.Compare(values[b], values[a])
	})

	n = min(max(n, 0), len(indices))
	topNKeys := make([]K, n)
	topNValues := make([]V, n)
	for i, idx := range indices[:n] {
		topNKeys[i] = keyAt(ioGroup, idx)
		topNValues[i] = values[idx]
	}

	return topNKeys, topNValues
}
