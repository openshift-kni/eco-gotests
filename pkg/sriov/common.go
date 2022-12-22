package sriov

import "fmt"

// undefinedCrdObjectErrString returns error message for an undefined CR.
func undefinedCrdObjectErrString(crName string) string {
	return fmt.Sprintf("can not redefine undefined %s", crName)
}
