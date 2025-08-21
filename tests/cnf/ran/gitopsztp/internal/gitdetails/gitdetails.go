package gitdetails

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/argocd"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/gitopsztp/internal/tsparams"
	. "github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/internal/ranparam"
	"k8s.io/apimachinery/pkg/util/wait"
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

// SetGitDetailsInArgoCd updates the git details for the provided Argo CD app.
func SetGitDetailsInArgoCd(
	appName string, gitDetails tsparams.ArgoCdGitDetails, waitForSync, syncMustBeValid bool) error {
	app, err := argocd.PullApplication(HubAPIClient, appName, ranparam.OpenshiftGitOpsNamespace)
	if err != nil {
		return err
	}

	appSource := app.Definition.Spec.Source
	if appSource.RepoURL == gitDetails.Repo &&
		appSource.TargetRevision == gitDetails.Branch &&
		appSource.Path == gitDetails.Path {
		glog.V(tsparams.LogLevel).Info("Provided git details already configured, no change required.")

		return nil
	}

	glog.V(tsparams.LogLevel).Infof("Updating argocd app %s to use git details %v", appName, gitDetails)

	appSource.RepoURL = gitDetails.Repo
	appSource.TargetRevision = gitDetails.Branch
	appSource.Path = gitDetails.Path

	_, err = app.Update(true)
	if err != nil {
		return err
	}

	if waitForSync {
		err = waitForArgoCdChangeToComplete(appName, syncMustBeValid, tsparams.ArgoCdChangeTimeout)
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdateArgoCdAppGitPath updates the git path in the specified Argo CD app, returning whether the git path exists and
// any error that occurred.
func UpdateArgoCdAppGitPath(appName, ztpTestPath string, syncMustBeValid bool) (bool, error) {
	gitDetails := tsparams.ArgoCdAppDetails[appName]
	testGitPath := JoinGitPaths([]string{
		gitDetails.Path,
		ztpTestPath,
	})

	if !DoesGitPathExist(gitDetails.Repo, gitDetails.Branch, testGitPath+tsparams.ZtpKustomizationPath) {
		return false, fmt.Errorf("git path '%s' could not be found", testGitPath)
	}

	gitDetails.Path = testGitPath
	err := SetGitDetailsInArgoCd(appName, gitDetails, true, syncMustBeValid)

	return true, err
}

// JoinGitPaths is used to join any combination of git strings but also avoiding double slashes.
func JoinGitPaths(inputs []string) string {
	// We want to preserve any existing double slashes but we don't want to add any between the input elements
	// To work around this we will use a special join character and a couple replacements
	special := "<<join>>"

	// Combine the inputs with the special join character
	result := strings.Join(
		inputs,
		special,
	)

	// Replace any special joins that have a slash prefix
	result = strings.ReplaceAll(result, "/"+special, "/")

	// Replace any special joins that have a slash suffix
	result = strings.ReplaceAll(result, special+"/", "/")

	// Finally replace any remaining special joins
	result = strings.ReplaceAll(result, special, "/")

	// The final result should never have double slashes between the joined elements
	// However if they already had any double slashes, e.g. "http://", they will be preserved
	return result
}

// DoesGitPathExist checks if the specified git url exists by sending an HTTP request to it.
func DoesGitPathExist(gitURL, gitBranch, gitPath string) bool {
	url := JoinGitPaths([]string{
		strings.Replace(gitURL, ".git", "", 1),
		"raw",
		gitBranch,
		gitPath,
	})

	glog.V(tsparams.LogLevel).Infof("Checking if git url '%s' exists", url)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Get(url)
	if err == nil && resp.StatusCode == 200 {
		glog.V(tsparams.LogLevel).Infof("found valid git url for '%s'", gitPath)

		return true
	}

	glog.V(tsparams.LogLevel).Infof("could not find valid url for '%s'", gitPath)

	return false
}

// waitForArgoCdChangeToComplete waits up to the specified timeout for the Argo CD configuration to be updated.
func waitForArgoCdChangeToComplete(appName string, syncMustBeValid bool, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(
		context.TODO(), tsparams.ArgoCdChangeInterval, timeout, true, func(ctx context.Context) (done bool, err error) {
			glog.V(tsparams.LogLevel).Infof("Checking if change to Argo CD app %s is complete", appName)

			app, err := argocd.PullApplication(HubAPIClient, appName, ranparam.OpenshiftGitOpsNamespace)
			if err != nil {
				return false, err
			}

			for i, condition := range app.Object.Status.Conditions {
				// If there are any conditions then it probably means theres a problem. By printing them
				// here we can make diagnosing a failing test easier.
				glog.V(tsparams.LogLevel).Infof("Argo CD app %s condition #%d: '%v'", appName, i, condition)
			}

			// The sync result may also have helpful information in the event of an error.
			operationState := app.Object.Status.OperationState
			if operationState != nil && operationState.SyncResult != nil {
				for i, resource := range operationState.SyncResult.Resources {
					if resource != nil {
						glog.V(tsparams.LogLevel).Infof("Argo CD app %s sync resource #%d: '%v'", appName, i, resource)
					}
				}
			}

			statusSource := app.Object.Status.Sync.ComparedTo.Source
			appSource := app.Object.Spec.Source

			if statusSource.RepoURL == appSource.RepoURL &&
				statusSource.TargetRevision == appSource.TargetRevision &&
				statusSource.Path == appSource.Path {
				if syncMustBeValid {
					return app.Object.Status.Sync.Status == "Synced", nil
				}

				return true, nil
			}

			return false, nil
		})
}
