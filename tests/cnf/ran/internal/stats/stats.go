package stats

import (
	"fmt"
	"math"
	"slices"
)

// Mean computes the arithmetic mean of the input array.
func Mean(input []float64) (float64, error) {
	if len(input) < 1 {
		return math.NaN(), fmt.Errorf("input array must have at least 1 element")
	}

	sum := 0.0
	for _, x := range input {
		sum += x
	}

	return sum / float64(len(input)), nil
}

// StdDev computes the population standard deviation of the input array.
func StdDev(input []float64) (float64, error) {
	if len(input) < 1 {
		return math.NaN(), fmt.Errorf("input array must have at least 1 element")
	}

	mean, _ := Mean(input)

	sum := 0.0
	for _, x := range input {
		sum += (x - mean) * (x - mean)
	}

	return math.Sqrt(sum / float64(len(input))), nil
}

// Median computes the median value of the input array.
func Median(input []float64) (float64, error) {
	if len(input) < 1 {
		return math.NaN(), fmt.Errorf("input array must have at least 1 element")
	}

	numElements := len(input)

	// sort a copy of the input array
	inputCopy := make([]float64, numElements)
	copy(inputCopy, input)

	slices.Sort(inputCopy)

	if numElements%2 == 1 {
		return inputCopy[numElements/2], nil
	}

	return (inputCopy[numElements/2] + inputCopy[numElements/2-1]) / 2, nil
}
