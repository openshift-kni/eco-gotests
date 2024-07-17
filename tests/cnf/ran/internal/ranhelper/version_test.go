package ranhelper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

//nolint:funlen
func TestIsVersionStringInRange(t *testing.T) {
	testCases := []struct {
		version        string
		minimum        string
		maximum        string
		expectedResult bool
		expectedError  error
	}{
		{
			version:        "4.16.0",
			minimum:        "4.10",
			maximum:        "",
			expectedResult: true,
			expectedError:  nil,
		},
		{
			version:        "4.16.0",
			minimum:        "",
			maximum:        "4.20",
			expectedResult: true,
			expectedError:  nil,
		},
		{
			version:        "4.16.0",
			minimum:        "4.10",
			maximum:        "4.20",
			expectedResult: true,
			expectedError:  nil,
		},
		{
			version:        "4.16.0",
			minimum:        "4.20",
			maximum:        "",
			expectedResult: false,
			expectedError:  nil,
		},
		{
			version:        "4.16.0",
			minimum:        "",
			maximum:        "4.10",
			expectedResult: false,
			expectedError:  nil,
		},
		{
			version:        "4.16.0",
			minimum:        "4.10",
			maximum:        "4.15",
			expectedResult: false,
			expectedError:  nil,
		},
		{
			version:        "4.16.0",
			minimum:        "4.0",
			maximum:        "5.0",
			expectedResult: false,
			expectedError:  nil,
		},
		{
			version:        "4.16.0",
			minimum:        "3.0",
			maximum:        "4.0",
			expectedResult: false,
			expectedError:  nil,
		},
		{
			version:        "4.16.0",
			minimum:        "invalid minimum",
			maximum:        "",
			expectedResult: false,
			expectedError:  fmt.Errorf("invalid minimum provided: 'invalid minimum'"),
		},
		{
			version:        "4.16.0",
			minimum:        "",
			maximum:        "invalid maximum",
			expectedResult: false,
			expectedError:  fmt.Errorf("invalid maximum provided: 'invalid maximum'"),
		},
		{
			version:        "",
			minimum:        "3.0",
			maximum:        "4.0",
			expectedResult: false,
			expectedError:  nil,
		},
		{
			version:        "",
			minimum:        "3.0",
			maximum:        "",
			expectedResult: true,
			expectedError:  nil,
		},
	}

	for _, testCase := range testCases {
		result, err := IsVersionStringInRange(testCase.version, testCase.minimum, testCase.maximum)

		assert.Equal(t, testCase.expectedResult, result)
		assert.Equal(t, testCase.expectedError, err)
	}
}

func TestGetInputIntegers(t *testing.T) {
	testCases := []struct {
		input  string
		output []int
	}{
		{
			input:  "4.16",
			output: []int{4, 16},
		},
		{
			input:  "4.16.0",
			output: []int{4, 16},
		},
		{
			input:  "invalid input",
			output: nil,
		},
		{
			// overflow the int64 type to get a range error
			input:  "4.99999999999999999999999",
			output: nil,
		},
	}

	for _, testCase := range testCases {
		output := getInputIntegers(testCase.input)

		assert.Equal(t, testCase.output, output)
	}
}
