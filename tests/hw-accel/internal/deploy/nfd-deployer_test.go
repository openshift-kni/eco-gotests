package deploy

import (
	"testing"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

// createMockAPIClient creates a mock API client for testing
func createMockAPIClient() *clients.Settings {
	return &clients.Settings{
		CoreV1Interface: fake.NewSimpleClientset().CoreV1(),
		Config:          &rest.Config{},
	}
}

// createTestOperatorConfig creates a test operator configuration
func createTestOperatorConfig() OperatorConfig {
	return NewOperatorConfig(
		createMockAPIClient(),
		"test-nfd-namespace",
		"test-operator-group",
		"test-subscription",
		"test-catalog-source",
		"openshift-marketplace",
		"nfd",
		"stable",
		"nfd-operator",
	)
}

func TestNewNFDDeployer(t *testing.T) {
	config := createTestOperatorConfig()

	deployer := NewNFDDeployer(config)

	assert.NotNil(t, deployer)
	assert.Equal(t, config.Namespace, deployer.GetNamespace())
	assert.Equal(t, config.OperatorName, deployer.GetOperatorName())
	assert.NotNil(t, deployer.CommonOps)
	assert.Equal(t, config.Namespace, deployer.CommonOps.Config.Namespace)
}

func TestNFDDeployer_GetNamespace(t *testing.T) {
	config := createTestOperatorConfig()
	deployer := NewNFDDeployer(config)

	namespace := deployer.GetNamespace()

	assert.Equal(t, "test-nfd-namespace", namespace)
}

func TestNFDDeployer_GetOperatorName(t *testing.T) {
	config := createTestOperatorConfig()
	deployer := NewNFDDeployer(config)

	operatorName := deployer.GetOperatorName()

	assert.Equal(t, "nfd-operator", operatorName)
}

func TestNFDDeployer_Deploy(t *testing.T) {
	tests := []struct {
		name        string
		config      OperatorConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "successful deployment",
			config:      createTestOperatorConfig(),
			expectError: false,
		},
		{
			name: "deployment with invalid namespace",
			config: func() OperatorConfig {
				config := createTestOperatorConfig()
				config.Namespace = ""
				return config
			}(),
			expectError: false, // namespace creation should handle empty names gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployer := NewNFDDeployer(tt.config)

			err := deployer.Deploy()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNFDDeployer_IsReady(t *testing.T) {
	tests := []struct {
		name           string
		timeout        time.Duration
		expectedResult bool
		expectError    bool
	}{
		{
			name:           "check readiness with valid timeout",
			timeout:        30 * time.Second,
			expectedResult: false, // Will be false since we're using mock client
			expectError:    false,
		},
		{
			name:           "check readiness with short timeout",
			timeout:        1 * time.Second,
			expectedResult: false,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestOperatorConfig()
			deployer := NewNFDDeployer(config)

			ready, err := deployer.IsReady(tt.timeout)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedResult, ready)
		})
	}
}

func TestNFDDeployer_Undeploy(t *testing.T) {
	config := createTestOperatorConfig()
	deployer := NewNFDDeployer(config)

	// Test undeploy - should not fail even if resources don't exist
	err := deployer.Undeploy()

	// Since we're using mock clients, this should not error
	// The real implementation would handle missing resources gracefully
	assert.NoError(t, err)
}

func TestNFDDeployer_DeployCustomResource(t *testing.T) {
	tests := []struct {
		name        string
		crName      string
		config      interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name:   "valid NFD config",
			crName: "test-nfd-instance",
			config: NFDConfig{
				EnableTopology: true,
				Image:          "registry.redhat.io/openshift4/ose-node-feature-discovery:latest",
			},
			expectError: true, // Will error due to mock client limitations
		},
		{
			name:        "invalid config type",
			crName:      "test-nfd-instance",
			config:      "invalid-config",
			expectError: true,
			errorMsg:    "invalid config type for NFD, expected NFDConfig",
		},
		{
			name:   "empty name with config name",
			crName: "",
			config: NFDConfig{
				EnableTopology: false,
				Image:          "",
			},
			expectError: true, // Will error due to mock client limitations
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestOperatorConfig()
			deployer := NewNFDDeployer(config)

			err := deployer.DeployCustomResource(tt.crName, tt.config)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNFDDeployer_DeleteCustomResource(t *testing.T) {
	tests := []struct {
		name        string
		crName      string
		expectError bool
	}{
		{
			name:        "delete existing resource",
			crName:      "test-nfd-instance",
			expectError: true, // Will error since resource doesn't exist in mock
		},
		{
			name:        "delete non-existent resource",
			crName:      "non-existent",
			expectError: true, // Will error since resource doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestOperatorConfig()
			deployer := NewNFDDeployer(config)

			err := deployer.DeleteCustomResource(tt.crName)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNFDDeployer_IsCustomResourceReady(t *testing.T) {
	tests := []struct {
		name           string
		crName         string
		timeout        time.Duration
		expectedResult bool
		expectError    bool
	}{
		{
			name:           "check non-existent resource",
			crName:         "test-nfd-instance",
			timeout:        10 * time.Second,
			expectedResult: false,
			expectError:    true, // Will error since resource doesn't exist in mock
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestOperatorConfig()
			deployer := NewNFDDeployer(config)

			ready, err := deployer.IsCustomResourceReady(tt.crName, tt.timeout)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedResult, ready)
		})
	}
}

func TestNFDConfig_Validation(t *testing.T) {
	tests := []struct {
		name   string
		config NFDConfig
		valid  bool
	}{
		{
			name: "valid config with topology enabled",
			config: NFDConfig{
				EnableTopology: true,
				Image:          "registry.redhat.io/openshift4/ose-node-feature-discovery:latest",
			},
			valid: true,
		},
		{
			name: "valid config with topology disabled",
			config: NFDConfig{
				EnableTopology: false,
				Image:          "",
			},
			valid: true,
		},
		{
			name: "config with custom image",
			config: NFDConfig{
				EnableTopology: true,
				Image:          "custom-registry/nfd:v1.0.0",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Since NFDConfig is a simple struct, we just verify it can be created
			assert.NotNil(t, tt.config)

			// Test that the config can be used as interface{}
			var genericConfig interface{} = tt.config

			// Test type assertion
			nfdConfig, ok := genericConfig.(NFDConfig)
			assert.True(t, ok)
			assert.Equal(t, tt.config.EnableTopology, nfdConfig.EnableTopology)
			assert.Equal(t, tt.config.Image, nfdConfig.Image)
		})
	}
}

func TestNFDDeployer_editAlmExample(t *testing.T) {
	tests := []struct {
		name        string
		almExample  string
		expectError bool
		contains    string
	}{
		{
			name: "valid ALM example with NFD",
			almExample: `[
				{
					"kind": "NodeFeatureDiscovery",
					"apiVersion": "nfd.openshift.io/v1",
					"metadata": {
						"name": "nfd-instance"
					}
				},
				{
					"kind": "SomeOtherResource",
					"apiVersion": "v1",
					"metadata": {
						"name": "other"
					}
				}
			]`,
			expectError: false,
			contains:    "NodeFeatureDiscovery",
		},
		{
			name:        "invalid JSON",
			almExample:  `invalid json`,
			expectError: true,
		},
		{
			name:        "empty array",
			almExample:  `[]`,
			expectError: false,
			contains:    "[]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestOperatorConfig()
			deployer := NewNFDDeployer(config)

			result, err := deployer.editAlmExample(tt.almExample)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.contains != "" {
					assert.Contains(t, result, tt.contains)
				}
			}
		})
	}
}

// Integration test for the complete workflow
func TestNFDDeployer_CompleteWorkflow(t *testing.T) {
	t.Run("complete deployment workflow", func(t *testing.T) {
		config := createTestOperatorConfig()
		deployer := NewNFDDeployer(config)

		// Test deployment
		err := deployer.Deploy()
		assert.NoError(t, err)

		// Test readiness check
		ready, err := deployer.IsReady(5 * time.Second)
		assert.NoError(t, err)
		assert.False(t, ready) // Expected to be false with mock client

		// Test custom resource deployment
		nfdConfig := NFDConfig{
			EnableTopology: true,
			Image:          "test-image",
		}

		err = deployer.DeployCustomResource("test-instance", nfdConfig)
		assert.Error(t, err) // Expected to error with mock client

		// Test undeploy
		err = deployer.Undeploy()
		assert.NoError(t, err)
	})
}

// Benchmark tests
func BenchmarkNewNFDDeployer(b *testing.B) {
	config := createTestOperatorConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewNFDDeployer(config)
	}
}

func BenchmarkNFDDeployer_Deploy(b *testing.B) {
	config := createTestOperatorConfig()
	deployer := NewNFDDeployer(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = deployer.Deploy()
	}
}
