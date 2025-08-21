package operator_test

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/api/v1beta1"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/assisted"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/namespace"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/pod"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/reportxml"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/service"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/assisted/ztp/internal/meets"
	. "github.com/rh-ecosystem-edge/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/assisted/ztp/operator/internal/tsparams"
	v1 "k8s.io/api/core/v1"
)

const (
	serverName          = "https-webserver"
	nsname              = "httpdtest"
	containerPort       = 8443
	httpdContainerImage = "registry.redhat.io/rhel8/httpd-24"
)

var (
	newAgentServiceConfig *assisted.AgentServiceConfigBuilder
	httpPodBuilder        *pod.Builder
	testOSImage           v1beta1.OSImage
	version               string = ZTPConfig.HubOCPXYVersion
)
var _ = Describe(
	"HttpWebserverSetup",
	Ordered,
	ContinueOnFailure,
	Label(tsparams.LabelHTTPWebserverSetup), func() {
		BeforeAll(func() {

			By("Validating that the environment is connected")
			connectionReq, msg := meets.HubConnectedRequirement()
			if !connectionReq {
				Skip(msg)
			}

			By("Creating httpd-test namespace")
			testNS, err := namespace.NewBuilder(HubAPIClient, nsname).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating namespace")

			By("Starting the https-webserver pod running an httpd container")
			httpPodBuilder = pod.NewBuilder(HubAPIClient, serverName, testNS.Definition.Name,
				httpdContainerImage).WithLabel("app", serverName)

			By("Adding an httpd container to the pod")
			httpPodBuilder.WithAdditionalContainer(&v1.Container{
				Name:    serverName,
				Image:   httpdContainerImage,
				Command: []string{"run-httpd"},
				Ports: []v1.ContainerPort{
					{
						ContainerPort: containerPort,
					},
				},
			})

			By("Creating the pod on the cluster")
			httpPodBuilder, err = httpPodBuilder.CreateAndWaitUntilRunning(time.Second * 180)
			Expect(err).ToNot(HaveOccurred(), "error creating pod on cluster")

			By("Create a service for the pod")
			serviceBuilder, err := service.NewBuilder(HubAPIClient, serverName, testNS.Definition.Name,
				map[string]string{"app": serverName}, v1.ServicePort{Port: containerPort, Protocol: "TCP"}).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating service")

			By("Downloading the new mirror")
			var imageName string
			for _, image := range ZTPConfig.HubAgentServiceConfg.Object.Spec.OSImages {
				if image.OpenshiftVersion == version {
					testOSImage = image
					splitURL := strings.Split(testOSImage.Url, "/")
					imageName = splitURL[len(splitURL)-1]
					_, err = httpPodBuilder.ExecCommand(
						[]string{"curl", "-k", image.Url, "-o", fmt.Sprintf("/var/www/html/%s", imageName)},
						serverName)

					Expect(err).ToNot(HaveOccurred(), "could not reach image url")

					break
				}
			}

			By("Deleting old agentserviceconfig")
			testOSImage.Url = fmt.Sprintf("https://%s.%s.svc.cluster.local:%d/%s",
				serviceBuilder.Object.Name, serviceBuilder.Object.Namespace, containerPort, imageName)
			err = ZTPConfig.HubAgentServiceConfg.DeleteAndWait(time.Second * 20)
			Expect(err).ToNot(HaveOccurred(), "could not delete agentserviceconfig")

		})

		It("Creates an agentserviceconfig with annotation and osImages pointing to new mirror",
			reportxml.ID("49577"), func() {

				newAgentServiceConfig = assisted.NewDefaultAgentServiceConfigBuilder(HubAPIClient).WithOSImage(testOSImage)
				newAgentServiceConfig.Definition.ObjectMeta.Annotations =
					map[string]string{"unsupported.agent-install.openshift.io/assisted-image-service-skip-verify-tls": "true"}
				_, err = newAgentServiceConfig.Create()
				Expect(err).ToNot(HaveOccurred(), "error while creating new agentserviceconfig")

				_, err = newAgentServiceConfig.WaitUntilDeployed(time.Second * 60)
				Expect(err).ToNot(HaveOccurred(), "error while deploying new agentserviceconfig")
			})

		AfterAll(func() {

			By("Deleting test namespace and pod")
			_, err = httpPodBuilder.DeleteAndWait(time.Second * 60)
			Expect(err).ToNot(HaveOccurred(), "could not delete pod")

			ns, err := namespace.Pull(HubAPIClient, nsname)
			Expect(err).ToNot(HaveOccurred(), "could not pull namespace")
			err = ns.DeleteAndWait(time.Second * 60)
			Expect(err).ToNot(HaveOccurred(), "could not delete namespace")

			By("Deleting the test agentserviceconfig")
			err = newAgentServiceConfig.DeleteAndWait(time.Second * 120)
			Expect(err).ToNot(HaveOccurred(), "could not delete agentserviceconfig")

			By("Restoring the original agentserviceconfig")
			_, err = ZTPConfig.HubAgentServiceConfg.Create()
			Expect(err).ToNot(HaveOccurred(), "could not reinstate original agentserviceconfig")

			_, err = ZTPConfig.HubAgentServiceConfg.WaitUntilDeployed(time.Second * 180)
			Expect(err).ToNot(HaveOccurred(), "error while deploying original agentserviceconfig")

		})

	})
