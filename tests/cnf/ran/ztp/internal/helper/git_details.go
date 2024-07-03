package helper

import (
	"github.com/openshift-kni/eco-goinfra/pkg/argocd"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranparam"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/ztp/internal/tsparams"
)

// GetArgoCdAppGitDetails initializes tsparams.ArgoCdAppDetails with the details for each of tsparams.ArgoCdApps.
func GetArgoCdAppGitDetails() error {
	for _, app := range tsparams.ArgoCdApps {
		argoCdApp, err := argocd.PullApplication(HubAPIClient, app, ranparam.OpenshiftGitOpsNamespace)
		if err != nil {
			return err
		}

		tsparams.ArgoCdAppDetails[app] = tsparams.ArgoCdGitDetails{
			Repo:   argoCdApp.Definition.Spec.Source.RepoURL,
			Branch: argoCdApp.Definition.Spec.Source.TargetRevision,
			Path:   argoCdApp.Definition.Spec.Source.Path,
		}
	}

	return nil
}
