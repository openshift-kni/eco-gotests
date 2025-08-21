package helper

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	pluginv1alpha1 "github.com/openshift-kni/oran-hwmgr-plugin/api/hwmgr-plugin/v1alpha1"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/configmap"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/ocm"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/oran"
	. "github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/oran/internal/tsparams"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	policiesv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewProvisioningRequest creates a ProvisioningRequest builder with templateVersion, setting all the required
// parameters and using the affix from RANConfig.
func NewProvisioningRequest(client *clients.Settings, templateVersion string) *oran.ProvisioningRequestBuilder {
	versionWithAffix := RANConfig.ClusterTemplateAffix + "-" + templateVersion
	prBuilder := oran.NewPRBuilder(client, tsparams.TestPRName, tsparams.ClusterTemplateName, versionWithAffix).
		WithTemplateParameter("nodeClusterName", RANConfig.Spoke1Name).
		WithTemplateParameter("oCloudSiteId", tsparams.OCloudSiteID).
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
		WithTemplateParameter("oCloudSiteId", tsparams.OCloudSiteID).
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

// WaitForNoncompliantImmutable waits up to timeout until one of the policies in namespace is NonCompliant and the
// message history shows it is due to an immutable field.
func WaitForNoncompliantImmutable(client *clients.Settings, namespace string, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(
		context.TODO(), 3*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			policies, err := ocm.ListPoliciesInAllNamespaces(client, runtimeclient.ListOptions{Namespace: namespace})
			if err != nil {
				glog.V(tsparams.LogLevel).Infof("Failed to list all policies in namespace %s: %v", namespace, err)

				return false, nil
			}

			for _, policy := range policies {
				if policy.Definition.Status.ComplianceState == policiesv1.NonCompliant {
					glog.V(tsparams.LogLevel).Infof("Policy %s in namespace %s is not compliant, checking history",
						policy.Definition.Name, policy.Definition.Namespace)

					details := policy.Definition.Status.Details
					if len(details) != 1 {
						continue
					}

					history := details[0].History
					if len(history) < 1 {
						continue
					}

					if strings.Contains(history[0].Message, tsparams.ImmutableMessage) {
						glog.V(tsparams.LogLevel).Infof("Policy %s in namespace %s is not compliant due to an immutable field",
							policy.Definition.Name, policy.Definition.Namespace)

						return true, nil
					}
				}
			}

			return false, nil
		})
}

// GetPolicyVersionForTemplate extracts the policy version from the ClusterInstance defaults ConfigMap for the provided
// ClusterTemplate.
func GetPolicyVersionForTemplate(client *clients.Settings, template *oran.ClusterTemplateBuilder) (string, error) {
	ciDefaultsCM, err := configmap.Pull(
		client, template.Object.Spec.Templates.ClusterInstanceDefaults, template.Definition.Namespace)
	if err != nil {
		glog.V(tsparams.LogLevel).Infof(
			"Failed to pull ClusterInstance defaults ConfigMap for template %s in namespace %s: %v",
			template.Definition.Name, template.Definition.Name, err)

		return "", err
	}

	ciDefaults, exists := ciDefaultsCM.Object.Data[tsparams.ClusterInstanceDefaultsKey]
	if !exists {
		return "", fmt.Errorf("clusterInstance defaults ConfigMap missing the defaults key %s",
			tsparams.ClusterInstanceDefaultsKey)
	}

	unmarshaledDefaults := make(map[string]any)
	err = yaml.Unmarshal([]byte(ciDefaults), &unmarshaledDefaults)

	if err != nil {
		return "", fmt.Errorf("failed to unmarshal ClusterInstance defaults data: %w", err)
	}

	extraLabels, exists := unmarshaledDefaults["extraLabels"]
	if !exists {
		return "", fmt.Errorf("clusterInstance defaults missing extraLabels key")
	}

	extraLabelsMap, typeOk := extraLabels.(map[string]any)
	if !typeOk {
		return "", fmt.Errorf("cannot assert extraLabels as map[string]any when its type is %T", extraLabels)
	}

	mclLabels, exists := extraLabelsMap["ManagedCluster"]
	if !exists {
		return "", fmt.Errorf("extraLabels map does not contain any labels for kind ManagedCluster")
	}

	mclLabelsMap, typeOk := mclLabels.(map[string]any)
	if !typeOk {
		return "", fmt.Errorf("cannot assert ManagedCluster labels as map[string]any when its type is %T", mclLabels)
	}

	policyVersion, exists := mclLabelsMap[tsparams.PolicySelectorLabel]
	if !exists {
		return "", fmt.Errorf("labels for kind ManagedCluster do not contain key %s", tsparams.PolicySelectorLabel)
	}

	policyVersionString, typeOk := policyVersion.(string)
	if !typeOk {
		return "", fmt.Errorf("cannot assert policy version label as string when its type is %T", policyVersion)
	}

	return policyVersionString, nil
}

// WaitForPolicyVersion waits up to timeout until all policies in namespace have version policyVersion, polling every 3
// seconds.
func WaitForPolicyVersion(client *clients.Settings, namespace, policyVersion string, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(
		context.TODO(), 3*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			policies, err := ocm.ListPoliciesInAllNamespaces(client, runtimeclient.ListOptions{Namespace: namespace})
			if err != nil {
				glog.V(tsparams.LogLevel).Infof("Failed to list all policies in namespace %s: %v", namespace, err)

				return false, nil
			}

			for _, policy := range policies {
				policySegments := strings.Split(policy.Definition.Name, ".")

				var policyName string

				// Generated policies will be of the format policyNamespace.policyname so we extract
				// only the name.
				if len(policySegments) == 0 {
					policyName = policy.Definition.Name
				} else {
					policyName = policySegments[len(policySegments)-1]
				}

				// All policy names should be of the format policyVersion-policyName.
				if !strings.HasPrefix(policyName, policyVersion) {
					return false, nil
				}
			}

			return true, nil
		})
}

// WaitForPRPolicyVersion waits up to timeout until all of the policies on the provided ProvisioningRequest have
// policyVersion.
func WaitForPRPolicyVersion(
	prBuilder *oran.ProvisioningRequestBuilder, policyVersion string, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(
		context.TODO(), 3*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			// Exists will update the Object with the latest ProvisioningRequest.
			if !prBuilder.Exists() {
				glog.V(tsparams.LogLevel).Infof("Failed to verify ProvisioningRequest %s exists", prBuilder.Definition.Name)

				return false, nil
			}

			for _, policyDetail := range prBuilder.Object.Status.Extensions.Policies {
				if !strings.HasPrefix(policyDetail.PolicyName, policyVersion) {
					glog.V(tsparams.LogLevel).Infof("Policy %s does not match expected policy version %s",
						policyDetail.PolicyName, policyVersion)

					return false, nil
				}
			}

			return true, nil
		})
}
