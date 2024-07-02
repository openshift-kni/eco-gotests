package file

import (
	"fmt"

	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranparam"
)

// TouchFile touches the file passed in parameter.
func TouchFile(path string) error {
	cmdToExec := fmt.Sprintf("sudo touch %s", path)

	return cluster.ExecCmd(raninittools.Spoke1APIClient, 3, ranparam.MasterNodeSelector, cmdToExec)
}

// DeleteFile deletes the file passed in parameter.
func DeleteFile(path string) error {
	cmdToExec := fmt.Sprintf("sudo rm %s", path)

	return fmt.Errorf("error deleting file %s, err=%w", path,
		cluster.ExecCmd(raninittools.Spoke1APIClient, 3, ranparam.MasterNodeSelector, cmdToExec))
}
