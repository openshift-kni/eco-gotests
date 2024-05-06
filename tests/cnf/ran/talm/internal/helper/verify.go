package helper

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/helper"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/talm/internal/tsparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IsVersionStringInRange checks if a version string is between a specified min and max value. All the string inputs to
// this function should be dot separated positive intergers, e.g. "1.0.0" or "4.10". Each string input must be at least
// two dot separarted integers but may also be 3 or more, though only the first two are compared.
func IsVersionStringInRange(version, minimum, maximum string) (bool, error) {
	versionValid, versionDigits := validateInputString(version)
	minimumValid, minimumDigits := validateInputString(minimum)
	maximumValid, maximumDigits := validateInputString(maximum)

	if !minimumValid {
		// Only accept invalid empty strings
		if minimum != "" {
			return false, fmt.Errorf("invalid minimum provided: '%s'", minimum)
		}

		// Assume the minimum digits are [0,0] for later comparison
		minimumDigits = []int{0, 0}
	}

	if !maximumValid {
		// Only accept invalid empty strings
		if maximum != "" {
			return false, fmt.Errorf("invalid maximum provided: '%s'", maximum)
		}

		// Assume the maximum digits are [math.MaxInt, math.MaxInt] for later comparison
		maximumDigits = []int{math.MaxInt, math.MaxInt}
	}

	// If the version was not valid then we need to check the min and max
	if !versionValid {
		// If no min or max was defined then return true
		if !minimumValid && !maximumValid {
			return true, nil
		}

		// Otherwise return whether the input maximum was an empty string or not
		return maximum == "", nil
	}

	// Otherwise the versions were valid so compare the digits
	for i := 0; i < 2; i++ {
		// The version bit should be between the minimum and maximum
		if versionDigits[i] < minimumDigits[i] || versionDigits[i] > maximumDigits[i] {
			return false, nil
		}
	}

	// At the end if we never returned then all the digits were in valid range
	return true, nil
}

// VerifyTalmIsInstalled checks that talm pod and container is present and that CGUs can be fetched.
func VerifyTalmIsInstalled() error {
	// Check for talm pods
	talmPods, err := pod.List(
		raninittools.Spoke1APIClient,
		tsparams.OpenshiftOperatorNamespace,
		metav1.ListOptions{LabelSelector: tsparams.TalmPodLabelSelector})
	if err != nil {
		return err
	}

	// Check if any pods exist
	if len(talmPods) == 0 {
		return errors.New("unable to find talm pod")
	}

	// Check each pod for the talm container
	for _, talmPod := range talmPods {
		if !helper.IsPodHealthy(talmPod) {
			return fmt.Errorf("talm pod %s is not healthy", talmPod.Definition.Name)
		}

		if !helper.DoesContainerExistInPod(talmPod, tsparams.TalmContainerName) {
			return errors.New("talm pod defined but talm container does not exist")
		}
	}

	return nil
}

// validateInputString validates that a string is at least two dot separated nonnegative integers.
func validateInputString(input string) (bool, []int) {
	versionSplits := strings.Split(input, ".")

	if len(versionSplits) < 2 {
		return false, []int{}
	}

	digits := []int{}

	for i := 0; i < 2; i++ {
		digit, err := strconv.Atoi(versionSplits[i])
		if err != nil || digit < 0 {
			return false, []int{}
		}

		digits = append(digits, digit)
	}

	return true, digits
}
