package msg

import "fmt"

// UndefinedCrdObjectErrString returns an error message for an undefined CR.
func UndefinedCrdObjectErrString(crName string) string {
	return fmt.Sprintf("can not redefine the undefined %s", crName)
}
