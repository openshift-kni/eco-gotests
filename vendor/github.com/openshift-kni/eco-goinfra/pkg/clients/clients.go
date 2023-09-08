package clients

import (
	"fmt"
	"log"
	"os"

	"github.com/golang/glog"
	"k8s.io/client-go/dynamic"

	argocdOperatorv1alpha1 "github.com/argoproj-labs/argocd-operator/api/v1alpha1"
	argocdScheme "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argocdClient "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned/typed/application/v1alpha1"
	bmhv1alpha1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	performanceV2 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/performanceprofile/v2"

	clientConfigV1 "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	v1security "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	ptpV1 "github.com/openshift/ptp-operator/pkg/client/clientset/versioned/typed/ptp/v1"
	olm2 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/scheme"

	olmv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/typed/operators/v1"
	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/typed/operators/v1alpha1"

	pkgManifestV1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
	clientPkgManifestV1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/client/clientset/versioned/typed/operators/v1"

	apiExt "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	appsV1Client "k8s.io/client-go/kubernetes/typed/apps/v1"
	networkV1Client "k8s.io/client-go/kubernetes/typed/networking/v1"
	rbacV1Client "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	netAttDefV1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	clientNetAttDefV1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/clientset/versioned/typed/k8s.cni.cncf.io/v1"
	srIovV1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"

	clientSrIovV1 "github.com/k8snetworkplumbingwg/sriov-network-operator/pkg/client/clientset/versioned/typed/sriovnetwork/v1"
	metalLbOperatorV1Beta1 "github.com/metallb/metallb-operator/api/v1beta1"

	clientMachineConfigV1 "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/typed/machineconfiguration.openshift.io/v1"
	metalLbV1Beta1 "go.universe.tf/metallb/api/v1beta1"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	nmstateV1alpha1 "github.com/nmstate/kubernetes-nmstate/api/v1alpha1"

	operatorV1 "github.com/openshift/api/operator/v1"
	hiveextV1Beta1 "github.com/openshift/assisted-service/api/hiveextension/v1beta1"
	agentInstallV1Beta1 "github.com/openshift/assisted-service/api/v1beta1"
	hiveV1 "github.com/openshift/hive/apis/hive/v1"
	moduleV1Beta1 "github.com/rh-ecosystem-edge/kernel-module-management/api/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"

	nvidiagpuv1 "github.com/NVIDIA/gpu-operator/api/v1"
	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
)

// Settings provides the struct to talk with relevant API.
type Settings struct {
	KubeconfigPath string
	coreV1Client.CoreV1Interface
	clientConfigV1.ConfigV1Interface
	clientMachineConfigV1.MachineconfigurationV1Interface
	networkV1Client.NetworkingV1Client
	appsV1Client.AppsV1Interface
	rbacV1Client.RbacV1Interface
	clientSrIovV1.SriovnetworkV1Interface
	Config *rest.Config
	runtimeClient.Client
	ptpV1.PtpV1Interface
	v1security.SecurityV1Interface
	olm.OperatorsV1alpha1Interface
	clientNetAttDefV1.K8sCniCncfIoV1Interface
	dynamic.Interface
	argocdClient.ArgoprojV1alpha1Interface
	olmv1.OperatorsV1Interface
	PackageManifestInterface clientPkgManifestV1.OperatorsV1Interface
}

// New returns a *Settings with the given kubeconfig.
func New(kubeconfig string) *Settings {
	var (
		config *rest.Config
		err    error
	)

	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	}

	if kubeconfig != "" {
		log.Printf("Loading kube client config from path %q", kubeconfig)
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		log.Print("Using in-cluster kube client config")
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		return nil
	}

	clientSet := &Settings{}
	clientSet.CoreV1Interface = coreV1Client.NewForConfigOrDie(config)
	clientSet.ConfigV1Interface = clientConfigV1.NewForConfigOrDie(config)
	clientSet.MachineconfigurationV1Interface = clientMachineConfigV1.NewForConfigOrDie(config)
	clientSet.AppsV1Interface = appsV1Client.NewForConfigOrDie(config)
	clientSet.SriovnetworkV1Interface = clientSrIovV1.NewForConfigOrDie(config)
	clientSet.NetworkingV1Client = *networkV1Client.NewForConfigOrDie(config)
	clientSet.PtpV1Interface = ptpV1.NewForConfigOrDie(config)
	clientSet.RbacV1Interface = rbacV1Client.NewForConfigOrDie(config)
	clientSet.OperatorsV1alpha1Interface = olm.NewForConfigOrDie(config)
	clientSet.K8sCniCncfIoV1Interface = clientNetAttDefV1.NewForConfigOrDie(config)
	clientSet.Interface = dynamic.NewForConfigOrDie(config)
	clientSet.OperatorsV1Interface = olmv1.NewForConfigOrDie(config)
	clientSet.PackageManifestInterface = clientPkgManifestV1.NewForConfigOrDie(config)
	clientSet.SecurityV1Interface = v1security.NewForConfigOrDie(config)
	clientSet.ArgoprojV1alpha1Interface = argocdClient.NewForConfigOrDie(config)

	clientSet.Config = config

	crScheme := runtime.NewScheme()
	err = SetScheme(crScheme)

	if err != nil {
		log.Print("Error to load apiClient scheme")

		return nil
	}

	clientSet.Client, err = runtimeClient.New(config, runtimeClient.Options{
		Scheme: crScheme,
	})

	if err != nil {
		log.Print("Error to create apiClient")

		return nil
	}

	clientSet.KubeconfigPath = kubeconfig

	return clientSet
}

// SetScheme returns mutated apiClient's scheme.
//
//nolint:funlen
func SetScheme(crScheme *runtime.Scheme) error {
	if err := scheme.AddToScheme(crScheme); err != nil {
		return err
	}

	if err := netAttDefV1.SchemeBuilder.AddToScheme(crScheme); err != nil {
		return err
	}

	if err := srIovV1.AddToScheme(crScheme); err != nil {
		return err
	}

	if err := mcv1.AddToScheme(crScheme); err != nil {
		return err
	}

	if err := apiExt.AddToScheme(crScheme); err != nil {
		return err
	}

	if err := metalLbOperatorV1Beta1.AddToScheme(crScheme); err != nil {
		return err
	}

	if err := metalLbV1Beta1.AddToScheme(crScheme); err != nil {
		return err
	}

	if err := performanceV2.AddToScheme(crScheme); err != nil {
		return err
	}

	if err := operatorV1.Install(crScheme); err != nil {
		return err
	}

	if err := olm2.AddToScheme(crScheme); err != nil {
		return err
	}

	if err := bmhv1alpha1.AddToScheme(crScheme); err != nil {
		return err
	}

	if err := hiveextV1Beta1.AddToScheme(crScheme); err != nil {
		return err
	}

	if err := hiveV1.AddToScheme(crScheme); err != nil {
		return err
	}

	if err := agentInstallV1Beta1.AddToScheme(crScheme); err != nil {
		return err
	}

	if err := moduleV1Beta1.AddToScheme(crScheme); err != nil {
		return err
	}

	if err := nvidiagpuv1.AddToScheme(crScheme); err != nil {
		return err
	}

	if err := nfdv1.AddToScheme(crScheme); err != nil {
		return err
	}

	if err := pkgManifestV1.AddToScheme(crScheme); err != nil {
		return err
	}

	if err := nmstatev1.AddToScheme(crScheme); err != nil {
		return err
	}

	if err := nmstateV1alpha1.AddToScheme(crScheme); err != nil {
		return err
	}

	if err := argocdOperatorv1alpha1.AddToScheme(crScheme); err != nil {
		return err
	}

	if err := argocdScheme.AddToScheme(crScheme); err != nil {
		return err
	}

	return nil
}

// GetAPIClient implements the cluster.APIClientGetter interface.
func (settings *Settings) GetAPIClient() (*Settings, error) {
	if settings == nil {
		glog.V(100).Infof("APIClient is nil")

		return nil, fmt.Errorf("APIClient cannot be nil")
	}

	return settings, nil
}
