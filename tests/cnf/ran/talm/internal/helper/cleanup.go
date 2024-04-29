package helper

import (
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/talm/internal/tsparams"
)

// DeleteTalmTestNamespace deletes the TALM test namespace.
func DeleteTalmTestNamespace() error {
	clusters := []*clients.Settings{raninittools.APIClient, raninittools.Spoke1APIClient, raninittools.Spoke2APIClient}

	for _, client := range clusters {
		if client != nil {
			err := namespace.NewBuilder(client, tsparams.TestNamespace).DeleteAndWait(5 * time.Minute)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
