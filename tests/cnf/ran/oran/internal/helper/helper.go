package helper

import (
	"fmt"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/oran"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/oran/internal/tsparams"
	pluginv1alpha1 "github.com/openshift-kni/oran-hwmgr-plugin/api/hwmgr-plugin/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewProvisioningRequest creates a ProvisioningRequest builder with templateVersion, setting all the required
// parameters and using the affix from RANConfig.
func NewProvisioningRequest(client *clients.Settings, templateVersion string) *oran.ProvisioningRequestBuilder {
	versionWithAffix := RANConfig.ClusterTemplateAffix + "-" + templateVersion
	prBuilder := oran.NewPRBuilder(client, tsparams.TestPRName, tsparams.ClusterTemplateName, versionWithAffix).
		WithTemplateParameter("nodeClusterName", RANConfig.Spoke1Name).
		WithTemplateParameter("oCloudSiteId", RANConfig.Spoke1Name).
		WithTemplateParameter("policyTemplateParameters", map[string]any{}).
		WithTemplateParameter("clusterInstanceParameters", map[string]any{
			"clusterName": RANConfig.Spoke1Name,
			"nodes": []map[string]any{{
				"hostName": RANConfig.Spoke1Hostname,
			}},
		})

	return prBuilder
}

// NewNoTemplatePR creates a ProvisioningRequest builder with templateVersion, following the schema for no
// HardwareTemplate. All required parameters and the affix are set from RANConfig. The BMC and network data are
// incorrect so that a ClusterInstance is generated but will not actually provision.
func NewNoTemplatePR(client *clients.Settings, templateVersion string) *oran.ProvisioningRequestBuilder {
	versionWithAffix := RANConfig.ClusterTemplateAffix + "-" + templateVersion
	prBuilder := oran.NewPRBuilder(client, tsparams.TestPRName, tsparams.ClusterTemplateName, versionWithAffix).
		WithTemplateParameter("nodeClusterName", RANConfig.Spoke1Name).
		WithTemplateParameter("oCloudSiteId", RANConfig.Spoke1Name).
		WithTemplateParameter("policyTemplateParameters", map[string]any{}).
		WithTemplateParameter("clusterInstanceParameters", map[string]any{
			"clusterName": RANConfig.Spoke1Name,
			"nodes": []map[string]any{{
				"hostName": RANConfig.Spoke1Hostname,
				// 192.0.2.0 is a reserved test address so we never accidentally use a valid IP.
				"bmcAddress": "redfish-VirtualMedia://192.0.2.0/redfish/v1/Systems/System.Embedded.1",
				"bmcCredentialsDetails": map[string]any{
					"username": tsparams.TestBase64Credential,
					"password": tsparams.TestBase64Credential,
				},
				"bootMACAddress": "01:23:45:67:89:AB",
				"nodeNetwork": map[string]any{
					"interfaces": []map[string]any{{
						"macAddress": "01:23:45:67:89:AB",
					}},
				},
			}},
		})

	return prBuilder
}

// GetValidDellHwmgr returns the first HardwareManager with AdaptorID dell-hwmgr and where condition Validation is True.
func GetValidDellHwmgr(client *clients.Settings) (*oran.HardwareManagerBuilder, error) {
	hwmgrs, err := oran.ListHardwareManagers(client, runtimeclient.ListOptions{
		Namespace: tsparams.HardwareManagerNamespace,
	})
	if err != nil {
		return nil, err
	}

	for _, hwmgr := range hwmgrs {
		if hwmgr.Definition.Spec.AdaptorID != pluginv1alpha1.SupportedAdaptors.Dell {
			continue
		}

		for _, condition := range hwmgr.Definition.Status.Conditions {
			if condition.Type == string(pluginv1alpha1.ConditionTypes.Validation) && condition.Status == metav1.ConditionTrue {
				return hwmgr, nil
			}
		}
	}

	return nil, fmt.Errorf("no valid HardwareManager with AdaptorID dell-hwmgr exists")
}
