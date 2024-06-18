package helper

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranparam"
	v1 "k8s.io/api/core/v1"
	"math"
	"strconv"
	"strings"
)

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

// IsPodHealthy returns true if a given pod is healthy, otherwise false.
func IsPodHealthy(pod *pod.Builder) bool {
	if pod.Object.Status.Phase == v1.PodRunning {
		// Check if running pod is ready
		if !isPodInCondition(pod, v1.PodReady) {
			glog.V(ranparam.LogLevel).Infof("pod condition is not Ready. Message: %s", pod.Object.Status.Message)

			return false
		}
	} else if pod.Object.Status.Phase != v1.PodSucceeded {
		// Pod is not running or completed.
		glog.V(ranparam.LogLevel).Infof("pod phase is %s. Message: %s", pod.Object.Status.Phase, pod.Object.Status.Message)

		return false
	}

	return true
}

// DoesContainerExistInPod checks if a given container exists in a given pod.
func DoesContainerExistInPod(pod *pod.Builder, containerName string) bool {
	containers := pod.Object.Status.ContainerStatuses

	for _, container := range containers {
		if container.Name == containerName {
			glog.V(ranparam.LogLevel).Infof("found %s container", containerName)

			return true
		}
	}

	return false
}

// isPodInCondition returns true if a given pod is in expected condition, otherwise false.
func isPodInCondition(pod *pod.Builder, condition v1.PodConditionType) bool {
	for _, c := range pod.Object.Status.Conditions {
		if c.Type == condition && c.Status == v1.ConditionTrue {
			return true
		}
	}

	return false
}
