package cleanup

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// CleanupNamespace cleanup specific namespace and delete it according to the flag config.
func CleanupNamespace(apiClient *clients.Settings, nsname string, toDelete bool, timeout time.Duration) error {
	glog.V(100).Infof("Scaling down namespace %s resources", nsname)

	deleteResources := []string{"deployment", "replicaset", "replicationcontroller", "statefulset"}

	deployments, err := deployment.List(apiClient, nsname, metav1.ListOptions{})
	glog.V(100).Infof("Scaling down deployment resources: %v", deployments)

	if err == nil {
		for _, depl := range deployments {
			depl, deletionErr := depl.WithReplicas(0).Update()

			if deletionErr != nil {
				return fmt.Errorf("failed to delete deployment %s in namespace %s",
					depl.Definition.Name, depl.Definition.Namespace)
			}
		}
		glog.V(100).Infof("no active deployments found in namespace %s", nsname)
	}

	replicaset, err := replicaset.List(apiClient, nsname, metav1.ListOptions{})
	glog.V(100).Infof("Scaling down replicaset resources: %v", replicaset)

	if err == nil {
		for _, depl := range deployments {
			depl, deletionErr := depl.WithReplicas(0).Update()

			if deletionErr != nil {
				return fmt.Errorf("failed to delete deployment %s in namespace %s",
					depl.Definition.Name, depl.Definition.Namespace)
			}
		}
		glog.V(100).Infof("no active deployments found in namespace %s", nsname)
	}

	return nil
}
