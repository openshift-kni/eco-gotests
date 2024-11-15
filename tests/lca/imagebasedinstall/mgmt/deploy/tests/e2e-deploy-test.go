package deploy_test

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/bmh"
	"github.com/openshift-kni/eco-goinfra/pkg/hive"
	"github.com/openshift-kni/eco-goinfra/pkg/ibi"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/ocm"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	hiveV1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/hive/api/v1"
	"github.com/openshift-kni/eco-goinfra/pkg/schemes/hive/api/v1/none"
	ibiv1alpha1 "github.com/openshift-kni/eco-goinfra/pkg/schemes/imagebasedinstall/api/hiveextensions/v1alpha1"
	"gopkg.in/yaml.v3"

	"github.com/openshift-kni/eco-goinfra/pkg/secret"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/mgmt/deploy/internal/networkconfig"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/mgmt/deploy/internal/tsparams"
	. "github.com/openshift-kni/eco-gotests/tests/lca/imagebasedinstall/mgmt/internal/mgmtinittools"

	v1 "k8s.io/api/core/v1"
)

const (
	interfaceName = "enp1s0"
)

var _ = Describe(
	"Performing image based installation",
	Ordered,
	Label(tsparams.LabelEndToEndDeployment), func() {
		var (
			ibiImageSetName string
		)
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
					Skip("Cluster must use static networking")
				}

				tsparams.ReporterNamespacesToDump[MGMTConfig.Cluster.Info.ClusterName] = "spoke namespace"

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

				for host, info := range MGMTConfig.Cluster.Info.Hosts {
					By("Create baremetalhost secret for " + host)
					_, err = secret.NewBuilder(
						APIClient, host, MGMTConfig.Cluster.Info.ClusterName, v1.SecretTypeOpaque).WithData(map[string][]byte{
						"username": []byte(info.BMC.User),
						"password": []byte(info.BMC.Password),
					}).Create()
					Expect(err).NotTo(HaveOccurred(), "error creating bmh secret")

					By("Create secret for static networking configuration")
					var nodeNetworkingConfig networkconfig.NetworkConfig

					if len(info.Network.Interfaces) != 1 {
						Skip("Cannot support nodes with more than one network interface")
					}

					for _, iface := range info.Network.Interfaces {

						nodeIPAddr, nodeIPNetwork, err := net.ParseCIDR(info.Network.Address.IPv4)
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
									IPv4: networkconfig.IPConfig{
										DHCP: false,
										Address: []networkconfig.IPAddress{
											{
												IP:           nodeIPAddr.String(),
												PrefixLength: strconv.Itoa(cidr),
											},
										},
										Enabled: true,
									},
									IPv6: networkconfig.IPConfig{
										Enabled: false,
									},
								},
							},
							Routes: networkconfig.Routes{
								Config: []networkconfig.RouteConfig{
									{
										Destination:      "0.0.0.0/0",
										NextHopAddress:   info.Network.Gateway.IPv4,
										NextHopInterface: interfaceName,
									},
								},
							},
							DNSResolver: networkconfig.DNSResolver{
								Config: networkconfig.DNSResolverConfig{
									Server: []string{
										info.Network.DNS.IPv4,
									},
								},
							},
						}
					}

					networkSecretContent, err := yaml.Marshal(&nodeNetworkingConfig)
					Expect(err).NotTo(HaveOccurred(), "error marshaling network configuration")

					_, err = secret.NewBuilder(APIClient, fmt.Sprintf("%s-nmstate-config", host),
						MGMTConfig.Cluster.Info.ClusterName, v1.SecretTypeOpaque).WithData(map[string][]byte{
						"nmstate": networkSecretContent,
					}).Create()
					Expect(err).NotTo(HaveOccurred(), "error creating network configuration secret")

					By("Create baremetalhost for " + host)
					hostBMH := bmh.NewBuilder(
						APIClient, host, MGMTConfig.Cluster.Info.ClusterName, info.BMC.URLv4, host, info.BMC.MACAddress, "UEFI")
					hostBMH.Definition.Spec.AutomatedCleaningMode = "disabled"
					hostBMH.Definition.Spec.PreprovisioningNetworkDataName = fmt.Sprintf("%s-nmstate-config", host)

					_, err = hostBMH.Create()
					Expect(err).NotTo(HaveOccurred(), "error creating baremetalhost")
				}

				var snoNodeName string

				for hostname := range MGMTConfig.Cluster.Info.Hosts {
					snoNodeName = hostname

					break
				}

				By("Create imageclusterinstall for IBI installation")
				imageClusterInstall := ibi.NewImageClusterInstallBuilder(
					APIClient, MGMTConfig.Cluster.Info.ClusterName, MGMTConfig.Cluster.Info.ClusterName, ibiImageSetName).
					WithClusterDeployment(MGMTConfig.Cluster.Info.ClusterName).WithHostname(snoNodeName).
					WithMachineNetwork(MGMTConfig.Cluster.Info.MachineCIDR.IPv4)

				if MGMTConfig.PublicSSHKey != "" {
					imageClusterInstall.WithSSHKey(MGMTConfig.PublicSSHKey)
				}

				if MGMTConfig.SeedClusterInfo.MirrorRegistryConfigured {
					imageClusterInstall.Definition.Spec.ImageDigestSources =
						MGMTConfig.SeedClusterInfo.MirrorConfig.Spec.ImageDigestMirrors
				}

				imageClusterInstall.Definition.Spec.BareMetalHostRef = &ibiv1alpha1.BareMetalHostReference{}
				imageClusterInstall.Definition.Spec.BareMetalHostRef.Name = snoNodeName
				imageClusterInstall.Definition.Spec.BareMetalHostRef.Namespace = MGMTConfig.Cluster.Info.ClusterName
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
				managedCluster := ocm.NewManagedClusterBuilder(APIClient, MGMTConfig.Cluster.Info.ClusterName).
					WithOptions(withHubAcceptsClient)

				if !managedCluster.Exists() {
					err = APIClient.Create(context.TODO(), managedCluster.Definition)
					if err == nil {
						managedCluster.Object = managedCluster.Definition
					}
				}
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
			})
	})

func withHubAcceptsClient(builder *ocm.ManagedClusterBuilder) (*ocm.ManagedClusterBuilder, error) {
	if builder == nil || builder.Definition == nil {
		return builder, fmt.Errorf("managedcluster builder and definition cannot be nil")
	}

	builder.Definition.Spec.HubAcceptsClient = true

	return builder, nil
}
