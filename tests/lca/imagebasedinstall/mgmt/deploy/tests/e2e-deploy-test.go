package deploy_test

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/bmh"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/hive"
	"github.com/openshift-kni/eco-goinfra/pkg/ibi"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/ocm"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-goinfra/pkg/schemes/assisted/api/v1beta1"
	hiveV1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/hive/api/v1"
	"github.com/openshift-kni/eco-goinfra/pkg/schemes/hive/api/v1/none"
	ibiv1alpha1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/imagebasedinstall/api/hiveextensions/v1alpha1"
	siteconfigv1alpha1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/siteconfig/v1alpha1"

	"github.com/openshift-kni/eco-goinfra/pkg/siteconfig"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift-kni/eco-goinfra/pkg/secret"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/mgmt/deploy/internal/networkconfig"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/mgmt/deploy/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/mgmt/internal/mgmtconfig"
	. "github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/mgmt/internal/mgmtinittools"
	"github.com/openshift-kni/eco-gotests/tests/lca/internal/brutil"
	"github.com/openshift-kni/eco-gotests/tests/lca/internal/installconfig"
	k8sScheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/mgmt/internal/mgmtparams"
	v1 "k8s.io/api/core/v1"
)

const (
	interfaceName = "enp1s0"

	extraManifestNamespace = "extranamespace"
	extraManifestConfigmap = "extra-configmap"

	extraManifestNamespaceConfigmapName = "extra-manifests-cm0"
	extraManifestConfigmapConfigmapName = "extra-manifests-cm1"

	caBundleConfigMapName = "ca-bundle-configmap"

	ibiClusterTemplateName = "ibi-cluster-templates-v1"
	ibiNodeTemplateName    = "ibi-node-templates-v1"

	ipv4AddrFamily          = "ipv4"
	ipv6AddrFamily          = "ipv6"
	reporterNamespaceToDump = "spoke namespace"
)

var (
	ibiImageSetName string

	spokeClient *clients.Settings
)

var _ = Describe(
	"Performing image based installation",
	Ordered,
	Label(tsparams.LabelEndToEndDeployment), func() {
		BeforeAll(func() {
			if MGMTConfig.Cluster == nil {
				Skip("Failed to collect cluster info")
			}

			if MGMTConfig.SeedClusterInfo == nil || MGMTConfig.SeedClusterInfo.SeedClusterOCPVersion == "" {
				Skip("Seed clusterinfo not supplied")
			}

			splitOCPVersion := strings.Split(MGMTConfig.SeedClusterInfo.SeedClusterOCPVersion, ".")

			if len(splitOCPVersion) < 2 {
				Skip("Could not determine short OCP version from seed cluster ocp version")
			}

			ibiImageSetName = strings.Join(splitOCPVersion[:2], ".")
		})

		It("through IBI operator is successful in a connected environment with static networking",
			reportxml.ID("76641"), func() {
				if !MGMTConfig.StaticNetworking {
					Skip("Cluster is deployed without static networking")
				}

				if MGMTConfig.SiteConfig {
					Skip("Cluster is deployed with siteconfig operator")
				}

				tsparams.ReporterNamespacesToDump[MGMTConfig.Cluster.Info.ClusterName] = reporterNamespaceToDump

				createIBIOResouces(ipv4AddrFamily)
			})

		It("through siteconfig operator is successful in an IPv6 proxy-enabled environment with DHCP networking",
			reportxml.ID("76642"), func() {
				if MGMTConfig.StaticNetworking {
					Skip("Cluster is deployed with static networking")
				}

				if !MGMTConfig.SiteConfig {
					Skip("Cluster is deployed without siteconfig operator")
				}

				if MGMTConfig.SeedClusterInfo.Proxy.HTTPProxy == "" && MGMTConfig.SeedClusterInfo.Proxy.HTTPSProxy == "" {
					Skip("Cluster not installed with proxy")
				}

				tsparams.ReporterNamespacesToDump[MGMTConfig.Cluster.Info.ClusterName] = reporterNamespaceToDump

				createSiteConfigResouces(ipv6AddrFamily)
			})

		It("through siteconfig operator is successful in an IPv4 environment with DHCP networking",
			reportxml.ID("no-testcase"), func() {
				if MGMTConfig.StaticNetworking {
					Skip("Cluster is deployed with static networking")
				}

				if !MGMTConfig.SiteConfig {
					Skip("Cluster is deployed without siteconfig operator")
				}

				if MGMTConfig.SeedClusterInfo.Proxy.HTTPProxy != "" || MGMTConfig.SeedClusterInfo.Proxy.HTTPSProxy != "" {
					Skip("Cluster installed with proxy")
				}

				tsparams.ReporterNamespacesToDump[MGMTConfig.Cluster.Info.ClusterName] = reporterNamespaceToDump

				createSiteConfigResouces(ipv4AddrFamily)
			})

		It("successfully creates extramanifests", reportxml.ID("76643"), func() {
			if !MGMTConfig.ExtraManifests {
				Skip("Cluster not configured with extra manifests")
			}

			By("Get spoke client")
			spokeClient = getSpokeClient()

			By("Pull namespace created by extra manifests")
			extraNamespace, err := namespace.Pull(spokeClient, extraManifestNamespace)
			Expect(err).NotTo(HaveOccurred(), "error pulling namespace created by extra manifests")

			By("Pull configmap created by extra manifests")
			extraConfigmap, err := configmap.Pull(spokeClient, extraManifestConfigmap, extraNamespace.Object.Name)
			Expect(err).NotTo(HaveOccurred(), "error pulling configmap created by extra manifests")
			Expect(len(extraConfigmap.Object.Data)).To(Equal(1), "error: got unexpected data in configmap")
			Expect(extraConfigmap.Object.Data["hello"]).To(Equal("world"),
				"error: extra manifest configmap has incorrect content")
		})

		It("successfully adds CA bundle", reportxml.ID("77795"), func() {
			if !MGMTConfig.CABundle {
				Skip("Cluster not configured with CA bundle")
			}

			By("Get spoke client")
			spokeClient = getSpokeClient()

			By("Validate adding a certificate by referencing a CA bundle", func() {
				execCmd := "grep -q qebox.redhat.com /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem"
				_, err := cluster.ExecCmdWithStdout(spokeClient, execCmd)
				Expect(err).ToNot(HaveOccurred(), "failed checking the ca bundle for expected entry: %s", err)
			})
		})

		It("successfully configured using FIPs", reportxml.ID("76644"), func() {
			if !MGMTConfig.SeedClusterInfo.HasFIPS {
				Skip("Cluster not using FIPS enabled seed image")
			}

			By("Get spoke client")
			spokeClient = getSpokeClient()

			By("Get spoke cluster-config configmap")
			clusterConifgMap, err := configmap.Pull(spokeClient, "cluster-config-v1", "kube-system")
			Expect(err).NotTo(HaveOccurred(), "error pulling cluster-config configmap from spoke cluster")

			installConfigData, ok := clusterConifgMap.Object.Data["install-config"]
			Expect(ok).To(BeTrue(), "error: cluster-config does not contain appropriate install-config key")

			spokeInstallConfig, err := installconfig.NewInstallConfigFromString(installConfigData)
			Expect(err).NotTo(HaveOccurred(), "error creating InstallConfig struct from configmap data")
			Expect(spokeInstallConfig.FIPS).To(BeTrue(),
				"error: installed spoke does not have expected FIPS value set in install-config")
		})
	})

func createSharedResources() {
	By("Create namespace for IBI installation")

	_, err := namespace.NewBuilder(APIClient, MGMTConfig.Cluster.Info.ClusterName).Create()
	Expect(err).NotTo(HaveOccurred(), "error creating namespace")

	By("Get pull secret from hub cluster")

	spokePullSecret, err := secret.Pull(APIClient, "pull-secret", "openshift-config")
	Expect(err).NotTo(HaveOccurred(), "error getting pull-secret from hub cluster")

	By("Create pull secret for spoke cluster")

	spokePullSecret.Definition.Name = MGMTConfig.Cluster.Info.ClusterName
	spokePullSecret.Definition.Namespace = MGMTConfig.Cluster.Info.ClusterName
	spokePullSecret.Definition.ResourceVersion = ""
	_, err = spokePullSecret.Create()
	Expect(err).NotTo(HaveOccurred(), "error creating spoke pull-secret")

	if MGMTConfig.ExtraManifests {
		By("Create namespace builder for extramanifests")

		extraNamespace := namespace.NewBuilder(APIClient, extraManifestNamespace)

		By("Create configmap for extra manifests namespace")

		extraNamespaceString, err := brutil.NewBackupRestoreObject(
			extraNamespace.Definition, k8sScheme.Scheme, v1.SchemeGroupVersion).String()
		Expect(err).NotTo(HaveOccurred(), "error creating configmap data for extramanifest namespace")
		_, err = configmap.NewBuilder(
			APIClient, extraManifestNamespaceConfigmapName, MGMTConfig.Cluster.Info.ClusterName).WithData(map[string]string{
			"00-namespace.yaml": extraNamespaceString,
		}).Create()
		Expect(err).NotTo(HaveOccurred(), "error creating configmap for extra manifests namespace")

		By("Create configmap builder for extramanifests")

		extraConfigmap := configmap.NewBuilder(
			APIClient, extraManifestConfigmap, extraManifestNamespace).WithData(map[string]string{
			"hello": "world",
		})

		By("Create configmap for extramanifests configmap")

		extraConfigmapString, err := brutil.NewBackupRestoreObject(
			extraConfigmap.Definition, k8sScheme.Scheme, v1.SchemeGroupVersion).String()
		Expect(err).NotTo(HaveOccurred(), "error creating configmap data for extramanifest configmap")
		_, err = configmap.NewBuilder(
			APIClient, extraManifestConfigmapConfigmapName, MGMTConfig.Cluster.Info.ClusterName).WithData(map[string]string{
			"01-configmap.yaml": extraConfigmapString,
		}).Create()
		Expect(err).NotTo(HaveOccurred(), "error creating configmap for extra manifests configmap")
	}

	if MGMTConfig.CABundle {
		By("Create configmap for CA bundle")

		_, err = configmap.NewBuilder(
			APIClient, caBundleConfigMapName, MGMTConfig.Cluster.Info.ClusterName).WithData(map[string]string{
			"tls-ca-bundle.pem": mgmtparams.CaBundleString,
		}).Create()
		Expect(err).NotTo(HaveOccurred(), "error creating configmap with CA bundle")
	}

	for host, info := range MGMTConfig.Cluster.Info.Hosts {
		By("Create baremetalhost secret for " + host)

		_, err = secret.NewBuilder(
			APIClient, host, MGMTConfig.Cluster.Info.ClusterName, v1.SecretTypeOpaque).WithData(map[string][]byte{
			"username": []byte(info.BMC.User),
			"password": []byte(info.BMC.Password),
		}).Create()
		Expect(err).NotTo(HaveOccurred(), "error creating bmh secret")
	}
}

//nolint:funlen
func createIBIOResouces(addressFamily string) {
	createSharedResources()

	var err error

	for host, info := range MGMTConfig.Cluster.Info.Hosts {
		By("Create baremetalhost for " + host)

		var bmcAddress string

		if addressFamily == ipv4AddrFamily {
			bmcAddress = info.BMC.URLv4
		} else {
			bmcAddress = info.BMC.URLv6
		}

		hostBMH := bmh.NewBuilder(
			APIClient, host, MGMTConfig.Cluster.Info.ClusterName, bmcAddress, host, info.BMC.MACAddress, "UEFI")
		hostBMH.Definition.Spec.AutomatedCleaningMode = "disabled"
		hostBMH.Definition.Spec.ExternallyProvisioned = true

		if MGMTConfig.StaticNetworking {
			nodeNetworkingConfig := createNetworkConfig(*MGMTConfig.Cluster, addressFamily)

			networkSecretContent, err := yaml.Marshal(&nodeNetworkingConfig)
			Expect(err).NotTo(HaveOccurred(), "error marshaling network configuration")

			_, err = secret.NewBuilder(APIClient, fmt.Sprintf("%s-nmstate-config", host),
				MGMTConfig.Cluster.Info.ClusterName, v1.SecretTypeOpaque).WithData(map[string][]byte{
				"nmstate": networkSecretContent,
			}).Create()
			Expect(err).NotTo(HaveOccurred(), "error creating network configuration secret")

			hostBMH.Definition.Spec.PreprovisioningNetworkDataName = fmt.Sprintf("%s-nmstate-config", host)
		}

		_, err = hostBMH.Create()
		Expect(err).NotTo(HaveOccurred(), "error creating baremetalhost")
	}

	var snoNodeName string

	for hostname := range MGMTConfig.Cluster.Info.Hosts {
		snoNodeName = hostname

		break
	}

	imageClusterInstall := ibi.NewImageClusterInstallBuilder(
		APIClient, MGMTConfig.Cluster.Info.ClusterName, MGMTConfig.Cluster.Info.ClusterName, ibiImageSetName).
		WithClusterDeployment(MGMTConfig.Cluster.Info.ClusterName).WithHostname(snoNodeName)

	if addressFamily == ipv4AddrFamily {
		imageClusterInstall.WithMachineNetwork(MGMTConfig.Cluster.Info.MachineCIDR.IPv4)
	} else {
		imageClusterInstall.WithMachineNetwork(MGMTConfig.Cluster.Info.MachineCIDR.IPv6)
	}

	if MGMTConfig.ExtraManifests {
		imageClusterInstall.WithExtraManifests(extraManifestNamespaceConfigmapName).
			WithExtraManifests(extraManifestConfigmapConfigmapName)
	}

	if MGMTConfig.CABundle {
		imageClusterInstall.WithCABundle(caBundleConfigMapName)
	}

	if MGMTConfig.PublicSSHKey != "" {
		imageClusterInstall.WithSSHKey(MGMTConfig.PublicSSHKey)
	}

	if MGMTConfig.SeedClusterInfo.MirrorRegistryConfigured {
		imageClusterInstall.Definition.Spec.ImageDigestSources =
			MGMTConfig.SeedClusterInfo.MirrorConfig.Spec.ImageDigestMirrors
	}

	imageClusterInstall.Definition.Spec.BareMetalHostRef = &ibiv1alpha1.BareMetalHostReference{
		Name:      snoNodeName,
		Namespace: MGMTConfig.Cluster.Info.ClusterName,
	}

	By("Create imageclusterinstall for IBI installation")

	_, err = imageClusterInstall.Create()
	Expect(err).NotTo(HaveOccurred(), "error creating imageclusterinstall")

	By("Create cluster deployment for IBI installation")

	_, err = hive.NewClusterDeploymentByInstallRefBuilder(
		APIClient, MGMTConfig.Cluster.Info.ClusterName,
		MGMTConfig.Cluster.Info.ClusterName, MGMTConfig.Cluster.Info.ClusterName,
		MGMTConfig.Cluster.Info.BaseDomain, hiveV1.ClusterInstallLocalReference{
			Group:   ibiv1alpha1.Group,
			Version: ibiv1alpha1.Version,
			Kind:    "ImageClusterInstall",
			Name:    MGMTConfig.Cluster.Info.ClusterName,
		}, hiveV1.Platform{
			None: &none.Platform{},
		}).
		WithPullSecret(MGMTConfig.Cluster.Info.ClusterName).Create()
	Expect(err).NotTo(HaveOccurred(), "error creating cluster deployment")

	By("Create managedcluster for IBI cluster")

	_, err = ocm.NewManagedClusterBuilder(APIClient, MGMTConfig.Cluster.Info.ClusterName).
		WithHubAcceptsClient(true).Create()
	Expect(err).NotTo(HaveOccurred(), "error creating managedcluster resource")

	Eventually(func() (bool, error) {
		imageClusterInstall.Object, err = imageClusterInstall.Get()
		if err != nil {
			return false, err
		}

		condition, err := imageClusterInstall.GetCompletedCondition()
		if err != nil {
			return false, err
		}

		return condition.Status == "True" && condition.Reason == ibiv1alpha1.InstallSucceededReason, nil

	}).WithTimeout(time.Minute*20).WithPolling(time.Second*5).Should(
		BeTrue(), "error waiting for imageclusterinstall to complete")
}

//nolint:funlen
func createSiteConfigResouces(addressFamily string) {
	createSharedResources()

	By("Find cluster template configmap")

	clusterTemplateConfigmap, err := configmap.ListInAllNamespaces(APIClient, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", ibiClusterTemplateName),
	})
	Expect(err).To(BeNil(), "error encountered when listing configmaps mactching cluster template name")
	Expect(len(clusterTemplateConfigmap)).To(Equal(1),
		"error: received unexpected configmap count: %d", len(clusterTemplateConfigmap))

	By("Find node template configmap")

	nodeTemplateConfigmap, err := configmap.ListInAllNamespaces(APIClient, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", ibiNodeTemplateName),
	})
	Expect(err).To(BeNil(), "error encountered when listing configmaps mactching node template name")
	Expect(len(nodeTemplateConfigmap)).To(Equal(1),
		"error: received unexpected configmap count: %d", len(nodeTemplateConfigmap))

	clusterInstanceBuilder := siteconfig.NewCIBuilder(
		APIClient, MGMTConfig.Cluster.Info.ClusterName, MGMTConfig.Cluster.Info.ClusterName).
		WithPullSecretRef(MGMTConfig.Cluster.Info.ClusterName).
		WithClusterTemplateRef(ibiClusterTemplateName, clusterTemplateConfigmap[0].Object.Namespace).
		WithBaseDomain(MGMTConfig.Cluster.Info.BaseDomain).
		WithClusterImageSetRef(ibiImageSetName).
		WithClusterName(MGMTConfig.Cluster.Info.ClusterName)

	if addressFamily == ipv4AddrFamily {
		clusterInstanceBuilder.WithMachineNetwork(MGMTConfig.Cluster.Info.MachineCIDR.IPv4)
	} else {
		clusterInstanceBuilder.WithMachineNetwork(MGMTConfig.Cluster.Info.MachineCIDR.IPv6)
	}

	if MGMTConfig.PublicSSHKey != "" {
		clusterInstanceBuilder.WithSSHPubKey(MGMTConfig.PublicSSHKey)
	}

	if MGMTConfig.ExtraManifests {
		clusterInstanceBuilder.WithExtraManifests(extraManifestNamespaceConfigmapName).
			WithExtraManifests(extraManifestConfigmapConfigmapName)
	}

	if MGMTConfig.CABundle {
		clusterInstanceBuilder.WithCABundle(caBundleConfigMapName)
	}

	if MGMTConfig.SeedClusterInfo.Proxy.HTTPProxy != "" || MGMTConfig.SeedClusterInfo.Proxy.HTTPSProxy != "" {
		clusterInstanceBuilder.WithProxy(&v1beta1.Proxy{
			HTTPProxy:  MGMTConfig.SeedClusterInfo.Proxy.HTTPProxy,
			HTTPSProxy: MGMTConfig.SeedClusterInfo.Proxy.HTTPSProxy,
			NoProxy:    MGMTConfig.SeedClusterInfo.Proxy.NOProxy,
		})
	}

	Expect(len(MGMTConfig.Cluster.Info.Hosts)).To(Equal(1), "error: can only support SNO deployments")

	for host, info := range MGMTConfig.Cluster.Info.Hosts {
		var bmcAddress string

		if addressFamily == ipv4AddrFamily {
			bmcAddress = info.BMC.URLv4
		} else {
			bmcAddress = info.BMC.URLv6
		}

		By("Add node entry for " + host)

		siteconfigNode := siteconfig.NewNodeBuilder(host, bmcAddress, info.BMC.MACAddress, host, ibiNodeTemplateName,
			nodeTemplateConfigmap[0].Object.Namespace).WithAutomatedCleaningMode("disabled")

		if MGMTConfig.StaticNetworking {
			if len(info.Network.Interfaces) != 1 {
				Skip("Cannot support nodes with more than one network interface")
			}

			nodeNetwork := &v1beta1.NMStateConfigSpec{}

			nodeNetworkingConfig := createNetworkConfig(*MGMTConfig.Cluster, addressFamily)

			for _, iface := range nodeNetworkingConfig.Interfaces {
				nodeNetwork.Interfaces = append(nodeNetwork.Interfaces, &v1beta1.Interface{
					Name:       iface.Name,
					MacAddress: iface.MACAddress,
				})
			}

			rawNetwork, err := yaml.Marshal(&nodeNetworkingConfig)
			Expect(err).NotTo(HaveOccurred(), "error marshaling network configuration")

			nodeNetwork.NetConfig = v1beta1.NetConfig{
				Raw: rawNetwork,
			}

			siteconfigNode.WithNodeNetwork(nodeNetwork)
		}

		nodeSpec, err := siteconfigNode.Generate()
		Expect(err).NotTo(HaveOccurred(), "error generating node spec for clusterinstance")

		clusterInstanceBuilder.WithNode(nodeSpec)
	}

	_, err = clusterInstanceBuilder.Create()
	Expect(err).NotTo(HaveOccurred(), "error creating clusterinstance")

	Eventually(func() (bool, error) {
		clusterInstanceBuilder.Object, err = clusterInstanceBuilder.Get()
		if err != nil {
			return false, err
		}

		for _, condition := range clusterInstanceBuilder.Object.Status.Conditions {
			if condition.Type == string(siteconfigv1alpha1.ClusterProvisioned) {
				return condition.Status == "True" && condition.Reason == string(siteconfigv1alpha1.Completed), nil

			}
		}

		return false, nil
	}).WithTimeout(time.Minute*30).WithPolling(time.Second*10).Should(
		BeTrue(), "error waiting for clusterinstance to finish provisioning")
}

func createNetworkConfig(config mgmtconfig.Cluster, addressFamily string) networkconfig.NetworkConfig {
	nodeNetworkingConfig := networkconfig.NetworkConfig{}

	Expect(len(config.Info.Hosts)).To(Equal(1), "error: can only support SNO deployments")

	for _, info := range MGMTConfig.Cluster.Info.Hosts {
		Expect(len(info.Network.Interfaces)).To(Equal(1), "error: can only support nodes with single network interface")

		for _, iface := range info.Network.Interfaces {
			var address, gateway, dns, destination string
			if addressFamily == ipv4AddrFamily {
				address = info.Network.Address.IPv4
				gateway = info.Network.Gateway.IPv4
				dns = info.Network.DNS.IPv4
				destination = "0.0.0.0/0"
			} else {
				address = info.Network.Address.IPv6
				gateway = info.Network.Gateway.IPv6
				dns = info.Network.DNS.IPv6
				destination = "::/0"
			}

			nodeIPAddr, nodeIPNetwork, err := net.ParseCIDR(address)
			Expect(err).NotTo(HaveOccurred(), "error gathering network info from provided address")

			cidr, _ := nodeIPNetwork.Mask.Size()

			nodeNetworkingConfig = networkconfig.NetworkConfig{
				Interfaces: []networkconfig.Interface{
					{
						Name:       interfaceName,
						Type:       "ethernet",
						State:      "up",
						Identifier: "mac-address",
						MACAddress: iface.MACAddress,
					},
				},
				Routes: networkconfig.Routes{
					Config: []networkconfig.RouteConfig{
						{
							Destination:      destination,
							NextHopAddress:   gateway,
							NextHopInterface: interfaceName,
						},
					},
				},
				DNSResolver: networkconfig.DNSResolver{
					Config: networkconfig.DNSResolverConfig{
						Server: []string{
							dns,
						},
					},
				},
			}

			if addressFamily == ipv4AddrFamily {
				nodeNetworkingConfig.Interfaces[0].IPv4 = networkconfig.IPConfig{
					DHCP: false,
					Address: []networkconfig.IPAddress{
						{
							IP:           nodeIPAddr.String(),
							PrefixLength: strconv.Itoa(cidr),
						},
					},
					Enabled: true,
				}
				nodeNetworkingConfig.Interfaces[0].IPv6 = networkconfig.IPConfig{
					Enabled: false,
				}
			} else {
				nodeNetworkingConfig.Interfaces[0].IPv6 = networkconfig.IPConfig{
					DHCP: false,
					Address: []networkconfig.IPAddress{
						{
							IP:           nodeIPAddr.String(),
							PrefixLength: strconv.Itoa(cidr),
						},
					},
					Enabled: true,
				}
				nodeNetworkingConfig.Interfaces[0].IPv4 = networkconfig.IPConfig{
					Enabled: false,
				}
			}
		}
	}

	return nodeNetworkingConfig
}

func getSpokeClient() *clients.Settings {
	if spokeClient == nil {
		By("Get spoke admin kubeconfig")

		adminKubeconfigSecret, err := secret.Pull(APIClient,
			fmt.Sprintf("%s-admin-kubeconfig", MGMTConfig.Cluster.Info.ClusterName), MGMTConfig.Cluster.Info.ClusterName)
		Expect(err).NotTo(HaveOccurred(), "error pulling spoke kubeconfig secret")

		adminKubeconfigContent, ok := adminKubeconfigSecret.Object.Data["kubeconfig"]
		Expect(ok).To(BeTrue(), "error checking for kubeconfig key from admin kubeconfig secret")

		By("Writing spoke admin kubeconfig to file")

		err = os.WriteFile("/tmp/spoke-kubeconfig", adminKubeconfigContent, 0755)
		Expect(err).NotTo(HaveOccurred(), "error writing spoke kubeconfig to file")

		spokeClient = clients.New("/tmp/spoke-kubeconfig")
		Expect(spokeClient).NotTo(BeNil(), "error creating client from spoke kubeconfig file")
	}

	return spokeClient
}
