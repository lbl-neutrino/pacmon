package monitors

import (
	"cmp"
	"math"
	"slices"
)

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
