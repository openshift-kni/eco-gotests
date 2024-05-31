package define

import (
	"fmt"

	"github.com/openshift-kni/eco-goinfra/pkg/rbac"
	"github.com/openshift-kni/eco-goinfra/pkg/serviceaccount"
	"github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	rbacv1 "k8s.io/api/rbac/v1"
)

// ModuleCRB returns the custom ClusterRoleBinding builder object.
func ModuleCRB(svcAccount serviceaccount.Builder, kmodName string) rbac.ClusterRoleBindingBuilder {
	crbName := fmt.Sprintf("%s-module-manager-rolebinding", kmodName)
	crb := rbac.NewClusterRoleBindingBuilder(inittools.APIClient,
		crbName,
		"system:openshift:scc:privileged",
		rbacv1.Subject{
			Name:      svcAccount.Object.Name,
			Kind:      "ServiceAccount",
			Namespace: svcAccount.Object.Namespace,
		})

	return *crb
}
