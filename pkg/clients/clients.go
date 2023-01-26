package clients

import (
	"os"

	bmhv1alpha1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"

	"github.com/golang/glog"
	performanceV2 "github.com/openshift-kni/numaresources-operator/api/numaresourcesoperator/v1alpha1"
	operatorV1 "github.com/openshift/api/operator/v1"
	clientConfigV1 "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	ptpV1 "github.com/openshift/ptp-operator/pkg/client/clientset/versioned/typed/ptp/v1"
	olm2 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/scheme"

	// nolint:lll
	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/typed/operators/v1alpha1"
	fecV2 "github.com/smart-edge-open/sriov-fec-operator/sriov-fec/api/v2"
	apiExt "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	appsV1Client "k8s.io/client-go/kubernetes/typed/apps/v1"
	networkV1Client "k8s.io/client-go/kubernetes/typed/networking/v1"
	rbacV1Client "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	netAttDefV1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	// nolint:lll
	clientNetAttDefV1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/clientset/versioned/typed/k8s.cni.cncf.io/v1"
	srIovV1 "github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1"

	// nolint:lll
	clientSrIovV1 "github.com/k8snetworkplumbingwg/sriov-network-operator/pkg/client/clientset/versioned/typed/sriovnetwork/v1"
	metalLbOperatorV1Beta1 "github.com/metallb/metallb-operator/api/v1beta1"

	// nolint:lll
	clientMachineConfigV1 "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned/typed/machineconfiguration.openshift.io/v1"
	metalLbV1Beta1 "go.universe.tf/metallb/api/v1beta1"

	"k8s.io/client-go/kubernetes/scheme"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"

	hiveextV1Beta1 "github.com/openshift/assisted-service/api/hiveextension/v1beta1"
	agentInstallV1Beta1 "github.com/openshift/assisted-service/api/v1beta1"
	hiveV1 "github.com/openshift/hive/apis/hive/v1"
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
	olm.OperatorsV1alpha1Interface
	clientNetAttDefV1.K8sCniCncfIoV1Interface
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
		glog.V(4).Infof("Loading kube client config from path %q", kubeconfig)
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		glog.V(4).Infof("Using in-cluster kube client config")
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

	clientSet.Config = config

	crScheme := setScheme()

	clientSet.Client, err = runtimeClient.New(config, runtimeClient.Options{
		Scheme: crScheme,
	})
	if err != nil {
		return nil
	}

	clientSet.KubeconfigPath = kubeconfig

	return clientSet
}

func setScheme() *runtime.Scheme {
	crScheme := runtime.NewScheme()

	if err := scheme.AddToScheme(crScheme); err != nil {
		panic(err)
	}

	if err := netAttDefV1.SchemeBuilder.AddToScheme(crScheme); err != nil {
		panic(err)
	}

	if err := srIovV1.AddToScheme(crScheme); err != nil {
		panic(err)
	}

	if err := mcv1.AddToScheme(crScheme); err != nil {
		panic(err)
	}

	if err := fecV2.AddToScheme(crScheme); err != nil {
		panic(err)
	}

	if err := apiExt.AddToScheme(crScheme); err != nil {
		panic(err)
	}

	if err := metalLbOperatorV1Beta1.AddToScheme(crScheme); err != nil {
		panic(err)
	}

	if err := metalLbV1Beta1.AddToScheme(crScheme); err != nil {
		panic(err)
	}

	if err := performanceV2.AddToScheme(crScheme); err != nil {
		panic(err)
	}

	if err := operatorV1.Install(crScheme); err != nil {
		panic(err)
	}

	if err := olm2.AddToScheme(crScheme); err != nil {
		panic(err)
	}

	if err := bmhv1alpha1.AddToScheme(crScheme); err != nil {
		panic(err)
	}

	if err := hiveextV1Beta1.AddToScheme(crScheme); err != nil {
		panic(err)
	}

	if err := hiveV1.AddToScheme(crScheme); err != nil {
		panic(err)
	}

	if err := agentInstallV1Beta1.AddToScheme(crScheme); err != nil {
		panic(err)
	}

	return crScheme
}
