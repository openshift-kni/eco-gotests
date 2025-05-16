package gitdetails

import (
	"fmt"

	"github.com/openshift-kni/eco-goinfra/pkg/argocd"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran-deployment/deploymenttypes/internal/tsparams"
)

// checkAppSpecSource checks that the app.Definition.Spec.Source tree exists
func checkAppSpecSource(app *argocd.ApplicationBuilder) error {
	if app == nil {
		return fmt.Errorf("application is nil")
	}

	if app.Definition == nil {
		return fmt.Errorf("application definition is nil")
	}

	if app.Definition.Spec.Source == nil {
		return fmt.Errorf("application source is nil")
	}

	return nil
}

// GetGitPath retrieves the git path from the provided Argo CD application. It returns an error if it encounters any nil
// pointers trying to access the source path.
func GetGitPath(app *argocd.ApplicationBuilder) (string, error) {

	err := checkAppSpecSource(app)
	if err != nil {
		return "", err
	}

	if len(app.Definition.Spec.Source.Path) <= 0 {
		return "", fmt.Errorf("application git Path is empty")
	}
	return app.Definition.Spec.Source.Path, nil
}

// GetGitRepoURL retrieves the git URL from the provided Argo CD application. It returns an error if it encounters any nil
// pointers trying to access the source path.
func GetGitRepoUrl(app *argocd.ApplicationBuilder) (string, error) {

	err := checkAppSpecSource(app)
	if err != nil {
		return "", err
	}

	if len(app.Definition.Spec.Source.Path) <= 0 {
		return "", fmt.Errorf("application git RepoURL is empty")
	}
	return app.Definition.Spec.Source.RepoURL, nil
}

// GetGitRepoURL retrieves the git revision from the provided Argo CD application. It returns an error if it encounters any nil
// pointers trying to access the source path.
func GetGitTargetRevision(app *argocd.ApplicationBuilder) (string, error) {

	err := checkAppSpecSource(app)
	if err != nil {
		return "", err
	}

	if len(app.Definition.Spec.Source.TargetRevision) <= 0 {
		return "", fmt.Errorf("application git TargetRevision is empty")
	}
	return app.Definition.Spec.Source.RepoURL, nil
}

// UpdateAndWaitForSync appends elements to the git path of the provided Argo CD application and waits for the source to
// be updated. The synced parameter indicates whether to wait for the application to be in a synced state or not.
func UpdateAndWaitForSync(app *argocd.ApplicationBuilder, synced bool, elements ...string) error {
	_, err := app.WithGitPathAppended(elements...).Update(true)
	if err != nil {
		return fmt.Errorf("failed to update the application: %w", err)
	}

	err = app.WaitForSourceUpdate(synced, tsparams.ArgoCdChangeTimeout)
	if err != nil {
		return fmt.Errorf("failed to wait for the application to sync: %w", err)
	}

	return nil
}
