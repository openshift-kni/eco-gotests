package brutil

import (
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/mgmt/internal/mgmtparams"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroScheme "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkloadBackup represents ibu-workload backup.
var WorkloadBackup = BackupRestoreObject{
	Scheme: veleroScheme.Scheme,
	GVR:    velerov1.SchemeGroupVersion,
	Object: &velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mgmtparams.LCAWorkloadName,
			Namespace: mgmtparams.LCAOADPNamespace,
		},
		Spec: velerov1.BackupSpec{
			IncludedNamespaces: []string{
				mgmtparams.LCAWorkloadName,
			},
			IncludedNamespaceScopedResources: []string{
				"deployments",
				"services",
				"routes",
			},
			ExcludedClusterScopedResources: []string{
				"persistentVolumes",
			},
			StorageLocation: "default",
		},
	},
}

// WorkloadRestore represents ibu-workload restore.
var WorkloadRestore = BackupRestoreObject{
	Scheme: veleroScheme.Scheme,
	GVR:    velerov1.SchemeGroupVersion,
	Object: &velerov1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mgmtparams.LCAWorkloadName,
			Namespace: mgmtparams.LCAOADPNamespace,
			Labels: map[string]string{
				"velero.io/storage-location": "default",
			},
		},
		Spec: velerov1.RestoreSpec{
			BackupName: mgmtparams.LCAWorkloadName,
		},
	},
}

// KlusterletBackup represents ibu klusterlet backup.
var KlusterletBackup = BackupRestoreObject{
	Scheme: veleroScheme.Scheme,
	GVR:    velerov1.SchemeGroupVersion,
	Object: &velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mgmtparams.LCAKlusterletName,
			Namespace: mgmtparams.LCAOADPNamespace,
		},
		Spec: velerov1.BackupSpec{
			IncludedNamespaces: []string{
				mgmtparams.LCAKlusterletNamespace,
			},
			IncludedClusterScopedResources: []string{
				"klusterlets.operator.open-cluster-management.io",
				"clusterclaims.cluster.open-cluster-management.io",
				"clusterroles.rbac.authorization.k8s.io",
				"clusterrolebindings.rbac.authorization.k8s.io",
			},
			IncludedNamespaceScopedResources: []string{
				"deployments",
				"serviceaccounts",
				"secrets",
			},
			StorageLocation: "default",
		},
	},
}

// KlusterletRestore represents ibu klusterlet restore.
var KlusterletRestore = BackupRestoreObject{
	Scheme: veleroScheme.Scheme,
	GVR:    velerov1.SchemeGroupVersion,
	Object: &velerov1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mgmtparams.LCAKlusterletName,
			Namespace: mgmtparams.LCAOADPNamespace,
			Labels: map[string]string{
				"velero.io/storage-location": "default",
			},
		},
		Spec: velerov1.RestoreSpec{
			BackupName: mgmtparams.LCAKlusterletName,
		},
	},
}
