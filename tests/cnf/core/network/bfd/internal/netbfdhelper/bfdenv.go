package netbfdhelper

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/bfd/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netenv"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodeIPs returns node IPs based on the label.
func NodeIPs(apiclient *clients.Settings, label string) ([]string, error) {
	var ips []string

	glog.V(90).Infof("Fetching Node IPs based on the label %v", label)

	ipsWithSubnet, err := nodes.ListExternalIPv4Networks(apiclient, metav1.ListOptions{LabelSelector: label})
	if err != nil {
		glog.V(90).Infof("Unable to fetch Node IPs. Hence returning empty slice")

		return ips, err
	}

	for _, ip := range ipsWithSubnet {
		ips = append(ips, strings.Split(ip, "/")[0])
	}

	return ips, nil
}

// DefineBFDConfig returns string which represents BFD config file peering to all given IP addresses.
func DefineBFDConfig(neighborsIPAddresses []string) string {
	bfdConfig := "bfd\n"

	glog.V(90).Infof("Generating BFD configuration for frr.conf")

	for _, ipAddress := range neighborsIPAddresses {
		bfdConfig += fmt.Sprintf(tsparams.PeerConfigTemplate, ipAddress)
	}

	bfdConfig += "!"

	return bfdConfig
}

// DefineBFDConfigMapData returns configMapData required for the test setup.
func DefineBFDConfigMapData(ipAddresses []string) map[string]string {
	configMapData := make(map[string]string)

	glog.V(90).Infof("Generating ConfigMap to start bfd daemon and define bfd peers for FRR")

	// run the bfd daemon on non-default port, so it won't collide with metallb's bfd.
	configMapData["bfdd.conf"] = "/usr/lib/frr/bfdd --bfdctl /tmp/bfdd.sock\n--dplaneaddr ipv4:127.0.0.1:50701\n"
	configMapData["daemons"] = tsparams.DaemonsFile
	configMapData["frr.conf"] = DefineBFDConfig(ipAddresses)
	configMapData["vtysh.conf"] = ""

	return configMapData
}

// DefineRolePolicy returns the PolicyRule required for creating Rule.
func DefineRolePolicy() rbacv1.PolicyRule {
	glog.V(90).Infof("Generating Policy Rule for the Role")

	rule := rbacv1.PolicyRule{
		APIGroups:     []string{"security.openshift.io"},
		ResourceNames: []string{"privileged"},
		Resources:     []string{"securitycontextconstraints"},
		Verbs:         []string{"use"},
	}

	return rule
}

// DefineRoleBindingSubject returns Subject required for creating RoleBinding.
func DefineRoleBindingSubject() rbacv1.Subject {
	glog.V(90).Infof("Generating Subject for the RoleBinding")

	subject := rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      "default",
		Namespace: tsparams.TestNamespace,
	}

	return subject
}

// DefineTestContainerSpec returns ContainerSpec for creating Daemonset.
func DefineTestContainerSpec() (corev1.Container, error) {
	glog.V(90).Infof("Generating Container Spec for Daemonset")

	container, err := pod.
		NewContainerBuilder(
			tsparams.AppName, netinittools.NetConfig.FrrImage, []string{"/sbin/tini", "--", "/usr/lib/frr/docker-start"}).
		WithSecurityCapabilities([]string{"NET_RAW", "SYS_ADMIN"}, true).
		WithVolumeMount(
			corev1.VolumeMount{Name: tsparams.WorkerConfigMapName, ReadOnly: true, MountPath: tsparams.MountPath}).
		GetContainerCfg()

	return *container, err
}

// DefineCmVolume returns Volume Spec for Daemonset.
func DefineCmVolume() corev1.Volume {
	glog.V(90).Infof("Generating Volume Spec for Daemonset")

	volume := corev1.Volume{
		Name: tsparams.WorkerConfigMapName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: tsparams.WorkerConfigMapName,
				},
			},
		},
	}

	return volume
}

// IsBFDStatusUp checks if the status of all the BFD peers of a given pod is up.
func IsBFDStatusUp(pod *pod.Builder, peers []string) error {
	if len(peers) == 0 {
		return fmt.Errorf("invalid input, peers size is 0")
	}

	glog.V(90).Infof("Checking if BFD is up for peers %v in pod %s", peers, pod.Object.Name)

	for _, peer := range peers {
		err := netenv.BFDHasStatus(pod, peer, "up")
		if err != nil {
			return err
		}
	}

	return nil
}
