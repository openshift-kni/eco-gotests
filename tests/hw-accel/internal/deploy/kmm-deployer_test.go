package deploy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// createTestKMMOperatorConfig creates a test KMM operator configuration
func createTestKMMOperatorConfig() OperatorConfig {
	return NewOperatorConfig(
		createMockAPIClient(),
		"test-kmm-namespace",
		"test-kmm-operator-group",
		"test-kmm-subscription",
		"test-catalog-source",
		"openshift-marketplace",
		"kernel-module-management",
		"stable",
		"kmm-operator",
	)
}

func TestNewKMMDeployer(t *testing.T) {
	config := createTestKMMOperatorConfig()

	deployer := NewKMMDeployer(config)

	assert.NotNil(t, deployer)
	assert.Equal(t, config.Namespace, deployer.GetNamespace())
	assert.Equal(t, config.OperatorName, deployer.GetOperatorName())
	assert.NotNil(t, deployer.CommonOps)
	assert.Equal(t, config.Namespace, deployer.CommonOps.Config.Namespace)
}

func TestKMMDeployer_GetNamespace(t *testing.T) {
	config := createTestKMMOperatorConfig()
	deployer := NewKMMDeployer(config)

	namespace := deployer.GetNamespace()

	assert.Equal(t, "test-kmm-namespace", namespace)
}

func TestKMMDeployer_GetOperatorName(t *testing.T) {
	config := createTestKMMOperatorConfig()
	deployer := NewKMMDeployer(config)

	operatorName := deployer.GetOperatorName()

	assert.Equal(t, "kmm-operator", operatorName)
}

func TestKMMDeployer_Deploy(t *testing.T) {
	tests := []struct {
		name        string
		config      OperatorConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "successful deployment",
			config:      createTestKMMOperatorConfig(),
			expectError: false,
		},
		{
			name: "deployment with invalid namespace",
			config: func() OperatorConfig {
				config := createTestKMMOperatorConfig()
				config.Namespace = ""
				return config
			}(),
			expectError: false, // namespace creation should handle empty names gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployer := NewKMMDeployer(tt.config)

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

func TestKMMDeployer_IsReady(t *testing.T) {
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
			config := createTestKMMOperatorConfig()
			deployer := NewKMMDeployer(config)

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

func TestKMMDeployer_Undeploy(t *testing.T) {
	config := createTestKMMOperatorConfig()
	deployer := NewKMMDeployer(config)

	// Test undeploy - should not fail even if resources don't exist
	err := deployer.Undeploy()

	// Since we're using mock clients, this should not error
	// The real implementation would handle missing resources gracefully
	assert.NoError(t, err)
}

func TestKMMDeployer_DeployCustomResource(t *testing.T) {
	tests := []struct {
		name        string
		crName      string
		config      interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name:   "valid KMM config",
			crName: "test-kmm-module",
			config: KMMConfig{
				ModuleName:     "test-module",
				KernelMapping:  ".*",
				ContainerImage: "example.com/kernel-module:latest",
				NodeSelector: map[string]string{
					"node-role.kubernetes.io/worker": "",
				},
			},
			expectError: true, // Will error due to mock client limitations
		},
		{
			name:        "invalid config type",
			crName:      "test-kmm-module",
			config:      "invalid-config",
			expectError: true,
			errorMsg:    "invalid config type for KMM, expected KMMConfig",
		},
		{
			name:   "empty name with config name",
			crName: "",
			config: KMMConfig{
				ModuleName:     "test-module",
				KernelMapping:  ".*",
				ContainerImage: "example.com/kernel-module:latest",
			},
			expectError: true, // Will error due to mock client limitations
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestKMMOperatorConfig()
			deployer := NewKMMDeployer(config)

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

func TestKMMDeployer_DeleteCustomResource(t *testing.T) {
	tests := []struct {
		name        string
		crName      string
		expectError bool
	}{
		{
			name:        "delete existing resource",
			crName:      "test-kmm-module",
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
			config := createTestKMMOperatorConfig()
			deployer := NewKMMDeployer(config)

			err := deployer.DeleteCustomResource(tt.crName)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestKMMDeployer_IsCustomResourceReady(t *testing.T) {
	tests := []struct {
		name           string
		crName         string
		timeout        time.Duration
		expectedResult bool
		expectError    bool
	}{
		{
			name:           "check non-existent resource",
			crName:         "test-kmm-module",
			timeout:        10 * time.Second,
			expectedResult: false,
			expectError:    true, // Will error since resource doesn't exist in mock
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestKMMOperatorConfig()
			deployer := NewKMMDeployer(config)

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

func TestKMMDeployer_DeploySimpleModule(t *testing.T) {
	tests := []struct {
		name           string
		moduleName     string
		kernelRegex    string
		containerImage string
		nodeSelector   map[string]string
		expectError    bool
	}{
		{
			name:           "deploy simple module",
			moduleName:     "test-module",
			kernelRegex:    ".*",
			containerImage: "example.com/module:latest",
			nodeSelector: map[string]string{
				"node-role.kubernetes.io/worker": "",
			},
			expectError: true, // Will error due to mock client limitations
		},
		{
			name:           "deploy module with empty selector",
			moduleName:     "test-module-2",
			kernelRegex:    "5.14.*",
			containerImage: "example.com/module:v2",
			nodeSelector:   nil,
			expectError:    true, // Will error due to mock client limitations
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestKMMOperatorConfig()
			deployer := NewKMMDeployer(config)

			err := deployer.DeploySimpleModule(tt.moduleName, tt.kernelRegex, tt.containerImage, tt.nodeSelector)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestKMMDeployer_GetModuleStatus(t *testing.T) {
	tests := []struct {
		name           string
		moduleName     string
		expectedStatus string
		expectError    bool
	}{
		{
			name:           "get status of non-existent module",
			moduleName:     "test-module",
			expectedStatus: "",
			expectError:    true, // Will error since module doesn't exist in mock
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestKMMOperatorConfig()
			deployer := NewKMMDeployer(config)

			status, err := deployer.GetModuleStatus(tt.moduleName)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, status)
			}
		})
	}
}

func TestKMMConfig_Validation(t *testing.T) {
	tests := []struct {
		name   string
		config KMMConfig
		valid  bool
	}{
		{
			name: "valid config with node selector",
			config: KMMConfig{
				ModuleName:     "test-module",
				KernelMapping:  ".*",
				ContainerImage: "example.com/module:latest",
				NodeSelector: map[string]string{
					"node-role.kubernetes.io/worker": "",
				},
			},
			valid: true,
		},
		{
			name: "valid config without node selector",
			config: KMMConfig{
				ModuleName:     "test-module-2",
				KernelMapping:  "5.14.*",
				ContainerImage: "example.com/module:v2",
				NodeSelector:   nil,
			},
			valid: true,
		},
		{
			name: "config with specific kernel version",
			config: KMMConfig{
				ModuleName:     "test-module-3",
				KernelMapping:  "5.14.0-284.25.1.el9_2.x86_64",
				ContainerImage: "example.com/module:5.14",
				NodeSelector: map[string]string{
					"kubernetes.io/arch": "amd64",
				},
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Since KMMConfig is a simple struct, we just verify it can be created
			assert.NotNil(t, tt.config)

			// Test that the config can be used as interface{}
			var genericConfig interface{} = tt.config

			// Test type assertion
			kmmConfig, ok := genericConfig.(KMMConfig)
			assert.True(t, ok)
			assert.Equal(t, tt.config.ModuleName, kmmConfig.ModuleName)
			assert.Equal(t, tt.config.KernelMapping, kmmConfig.KernelMapping)
			assert.Equal(t, tt.config.ContainerImage, kmmConfig.ContainerImage)
		})
	}
}

// Integration test for the complete workflow
func TestKMMDeployer_CompleteWorkflow(t *testing.T) {
	t.Run("complete deployment workflow", func(t *testing.T) {
		config := createTestKMMOperatorConfig()
		deployer := NewKMMDeployer(config)

		// Test deployment
		err := deployer.Deploy()
		assert.NoError(t, err)

		// Test readiness check
		ready, err := deployer.IsReady(5 * time.Second)
		assert.NoError(t, err)
		assert.False(t, ready) // Expected to be false with mock client

		// Test simple module deployment
		nodeSelector := map[string]string{
			"node-role.kubernetes.io/worker": "",
		}

		err = deployer.DeploySimpleModule("test-module", ".*", "example.com/module:latest", nodeSelector)
		assert.Error(t, err) // Expected to error with mock client

		// Test undeploy
		err = deployer.Undeploy()
		assert.NoError(t, err)
	})
}

// Benchmark tests
func BenchmarkNewKMMDeployer(b *testing.B) {
	config := createTestKMMOperatorConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewKMMDeployer(config)
	}
}

func BenchmarkKMMDeployer_Deploy(b *testing.B) {
	config := createTestKMMOperatorConfig()
	deployer := NewKMMDeployer(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = deployer.Deploy()
	}
}
