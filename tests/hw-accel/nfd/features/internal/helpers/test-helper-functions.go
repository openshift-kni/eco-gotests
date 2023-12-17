package helpers

import (
	"fmt"
	"strings"

	ts "github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/features/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/internal/search"
)

// CheckLabelsExist check if each node contains the required label.
func CheckLabelsExist(nodelabels map[string][]string, labelsToSearch, blackList []string, nodeName string) error {
	labelList := ts.DefaultBlackList

	if blackList != nil || len(blackList) != 0 {
		labelList = blackList
	}

	allFeatures := strings.Join(nodelabels[nodeName], ",")

	if len(allFeatures) == 0 {
		return fmt.Errorf("node feature labels should be greater than zero")
	}

	for _, featurelabel := range labelsToSearch {
		if search.StringInSlice(featurelabel, labelList) {
			continue
		}

		if !strings.Contains(allFeatures, fmt.Sprintf("%s=", featurelabel)) {
			return fmt.Errorf("label %s not found in node %s", featurelabel, nodeName)
		}
	}

	return nil
}
