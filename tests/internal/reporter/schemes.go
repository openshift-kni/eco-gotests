package reporter

import (
	nvidiagpuv1 "github.com/NVIDIA/gpu-operator/api/v1"
	multinetpolicyapiv1beta1 "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1beta1"
	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	sriovv1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"
	bmhv1alpha1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/api/v1alpha1"
	cguv1alpha1 "github.com/openshift-kni/cluster-group-upgrades-operator/pkg/api/clustergroupupgrades/v1alpha1"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/schemes/argocd/argocdoperator"
	argocdv1alpha1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/argocd/argocdtypes/v1alpha1"
	hiveextv1beta1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/assisted/api/hiveextension/v1beta1"
	siteconfigv1alpha1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/siteconfig/v1alpha1"

	agentinstallv1beta1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/assisted/api/v1beta1"
	hivev1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/assisted/hive/api/v1"
	ibiv1alpha1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/imagebasedinstall/api/hiveextensions/v1alpha1"
	olmv1alpha1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/olm/operators/v1alpha1"
	lcav1 "github.com/openshift-kni/lifecycle-agent/api/imagebasedupgrade/v1"
	configv1 "github.com/openshift/api/config/v1"
	imageregistryv1 "github.com/openshift/api/imageregistry/v1"
	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	performanceprofilev2 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/performanceprofile/v2"
	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	mcmv1beta1 "github.com/rh-ecosystem-edge/kernel-module-management/api-hub/v1beta1"
	modulev1beta1 "github.com/rh-ecosystem-edge/kernel-module-management/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	policiesv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
	policiesv1beta1 "open-cluster-management.io/governance-policy-propagator/api/v1beta1"
	placementrulev1 "open-cluster-management.io/multicloud-operators-subscription/pkg/apis/apps/placementrule/v1"
)

var reporterSchemes = []clients.SchemeAttacher{
	clients.SetScheme,
	hivev1.AddToScheme,
	hiveextv1beta1.AddToScheme,
	agentinstallv1beta1.AddToScheme,
	bmhv1alpha1.AddToScheme,
	configv1.Install,
	nadv1.AddToScheme,
	nmstatev1.AddToScheme,
	nmstatev1alpha1.AddToScheme,
	sriovv1.AddToScheme,
	machineconfigv1.Install,
	performanceprofilev2.AddToScheme,
	multinetpolicyapiv1beta1.AddToScheme,
	policiesv1.AddToScheme,
	placementrulev1.AddToScheme,
	imageregistryv1.Install,
	argocdoperator.AddToScheme,
	argocdv1alpha1.AddToScheme,
	cguv1alpha1.AddToScheme,
	olmv1alpha1.AddToScheme,
	policiesv1beta1.AddToScheme,
	mcmv1beta1.AddToScheme,
	modulev1beta1.AddToScheme,
	nfdv1.AddToScheme,
	nvidiagpuv1.AddToScheme,
	lcav1.AddToScheme,
	ibiv1alpha1.AddToScheme,
	siteconfigv1alpha1.AddToScheme,
}

func setReporterSchemes(scheme *runtime.Scheme) error {
	for _, schemeAttacher := range reporterSchemes {
		if err := schemeAttacher(scheme); err != nil {
			return err
		}
	}

	return nil
}
