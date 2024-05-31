package helper

import (
	"errors"
	"math"
	"sort"
)

// min returns the minimum value of the input array.
func min(input []float64) (float64, error) {
	if len(input) < 1 {
		return math.NaN(), errors.New("input array must have at least 1 element")
	}

	min := input[0]
	for _, x := range input {
		if x < min {
			min = x
		}
	}

	return min, nil
}

// max returns the maximum value of the input array.
func max(input []float64) (float64, error) {
	if len(input) < 1 {
		return math.NaN(), errors.New("input array must have at least 1 element")
	}

	max := input[0]
	for _, x := range input {
		if x > max {
			max = x
		}
	}

	return max, nil
}

// mean computes the mean value of the input array.
func mean(input []float64) (float64, error) {
	if len(input) < 1 {
		return math.NaN(), errors.New("input array must have at least 1 element")
	}

	sum := 0.0
	for _, x := range input {
		sum += x
	}

	return sum / float64(len(input)), nil
}

// stdDev computes the population standard deviation of the input array.
func stdDev(input []float64) (float64, error) {
	if len(input) < 1 {
		return math.NaN(), errors.New("input array must have at least 1 element")
	}

	mean, _ := mean(input)

	sum := 0.0
	for _, x := range input {
		sum += (x - mean) * (x - mean)
	}

	return math.Sqrt(sum / float64(len(input))), nil
}

// median computes the median value of the input array.
func median(input []float64) (float64, error) {
	if len(input) < 1 {
		return math.NaN(), errors.New("input array must have at least 1 element")
	}

	numElements := len(input)

	// sort a copy of the input array
	inputCopy := make([]float64, numElements)
	copy(inputCopy, input)

	sort.Float64s(inputCopy)

	if numElements%2 == 1 {
		return inputCopy[numElements/2], nil
	}

	return (inputCopy[numElements/2] + inputCopy[numElements/2-1]) / 2, nil
}
