# Interface-Based Operator Deployment Framework

This package provides a clean, extensible interface-based framework for deploying OpenShift operators and their custom resources.

## Architecture Overview

The framework is built around two main interfaces:

- **`OperatorDeployer`**: Handles operator lifecycle (deploy, undeploy, readiness checks)
- **`CustomResourceDeployer`**: Handles custom resource operations

## Key Components

### Interfaces (`interfaces.go`)
- `OperatorDeployer`: Common operator operations
- `CustomResourceDeployer`: CR-specific operations
- `OperatorConfig`: Shared configuration structure
- `BaseOperatorDeployer`: Common functionality base

### Common Utilities (`common.go`)
- `CommonDeploymentOps`: Shared deployment operations
- Namespace management
- OLM resource handling (OperatorGroup, Subscription, CSV)
- Deployment readiness checks

### Operator-Specific Implementations
- **`nfd-deployer.go`**: Node Feature Discovery operator
- **`kmm-deployer.go`**: Kernel Module Management operator

## Usage Examples

### Basic NFD Deployment
```go
// Create configuration
config := NewOperatorConfig(
    apiClient,
    "openshift-nfd",
    "nfd-operator-group", 
    "nfd-subscription",
    "certified-operators",
    "openshift-marketplace",
    "nfd",
    "stable",
    "nfd-operator",
)

// Create deployer
nfdDeployer := NewNFDDeployer(config)

// Deploy operator
err := nfdDeployer.Deploy()

// Wait for readiness
ready, err := nfdDeployer.IsReady(5 * time.Minute)

// Deploy custom resource
nfdConfig := NFDConfig{
    EnableTopology: true,
    Image: "registry.redhat.io/openshift4/ose-node-feature-discovery:latest",
}
err = nfdDeployer.DeployCustomResource("nfd-instance", nfdConfig)
```

### Basic KMM Deployment
```go
// Create configuration
config := NewOperatorConfig(
    apiClient,
    "openshift-kmm",
    "kmm-operator-group",
    "kmm-subscription", 
    "certified-operators",
    "openshift-marketplace",
    "kernel-module-management",
    "stable",
    "kmm-operator",
)

// Create deployer
kmmDeployer := NewKMMDeployer(config)

// Deploy operator
err := kmmDeployer.Deploy()

// Deploy simple module
nodeSelector := map[string]string{
    "node-role.kubernetes.io/worker": "",
}
err = kmmDeployer.DeploySimpleModule(
    "example-module",
    ".*",
    "example.com/kernel-module:latest", 
    nodeSelector,
)
```

### Polymorphic Usage
```go
// Use interface for uniform handling
deployers := []OperatorDeployer{nfdDeployer, kmmDeployer}

for _, deployer := range deployers {
    fmt.Printf("Deploying %s in %s\n", 
        deployer.GetOperatorName(), 
        deployer.GetNamespace())
    
    if err := deployer.Deploy(); err != nil {
        log.Errorf("Failed to deploy %s: %v", deployer.GetOperatorName(), err)
        continue
    }
    
    ready, err := deployer.IsReady(5 * time.Minute)
    if err != nil || !ready {
        log.Errorf("Operator %s not ready: %v", deployer.GetOperatorName(), err)
    }
}
```

## Benefits Over Original Design

### ✅ **Separation of Concerns**
- Each deployer handles one operator type
- Common functionality is shared via composition
- Clean interface contracts

### ✅ **Extensibility** 
- Easy to add new operators (SRIOV, ODF, etc.)
- Uniform patterns across all operators
- Interface-based polymorphism

### ✅ **Testability**
- Easy mocking via interfaces
- Comprehensive unit test coverage
- Isolated testing of components

### ✅ **Maintainability**
- Clear code organization
- Reduced duplication
- Type-safe operations

### ✅ **Reusability**
- Common patterns abstracted
- Consistent API across operators
- Composable design

## Testing

The framework includes comprehensive tests:

- **Unit Tests**: `*_test.go` files for each component
- **Integration Tests**: Complete workflow testing
- **Benchmark Tests**: Performance validation
- **Mock Support**: Easy testing with fake clients

Run tests:
```bash
go test ./tests/hw-accel/internal/deploy/...
```

## Configuration Management

### NFD Configuration
```go
type NFDConfig struct {
    EnableTopology bool   // Enable topology updater
    Image          string // Custom NFD image (optional)
}
```

### KMM Configuration  
```go
type KMMConfig struct {
    ModuleName     string            // Module name
    KernelMapping  string            // Kernel version regex
    ContainerImage string            // Module container image
    NodeSelector   map[string]string // Node selection criteria
}
```

## Adding New Operators

To add a new operator:

1. **Create deployer file**: `new-operator-deployer.go`
2. **Implement interfaces**: `OperatorDeployer` and `CustomResourceDeployer`
3. **Define config struct**: Operator-specific configuration
4. **Create tests**: `new-operator-deployer_test.go`
5. **Update examples**: Add usage examples

Example structure:
```go
type NewOperatorDeployer struct {
    BaseOperatorDeployer
    CommonOps *CommonDeploymentOps
}

func (n *NewOperatorDeployer) Deploy() error { /* implementation */ }
func (n *NewOperatorDeployer) IsReady(timeout time.Duration) (bool, error) { /* implementation */ }
func (n *NewOperatorDeployer) Undeploy() error { /* implementation */ }
func (n *NewOperatorDeployer) DeployCustomResource(name string, config interface{}) error { /* implementation */ }
func (n *NewOperatorDeployer) DeleteCustomResource(name string) error { /* implementation */ }
func (n *NewOperatorDeployer) IsCustomResourceReady(name string, timeout time.Duration) (bool, error) { /* implementation */ }
```

## Migration from Original Code

The original `deploy-nfd.go` can be gradually migrated:

1. **Replace direct usage**:
   ```go
   // Old
   nfdResource := NewNfdAPIResource(...)
   err := nfdResource.DeployNfd(...)
   
   // New
   nfdDeployer := NewNFDDeployer(config)
   err := nfdDeployer.Deploy()
   ```

2. **Update test code** to use new interfaces
3. **Leverage polymorphism** for multi-operator scenarios
4. **Remove old code** once migration is complete

This interface-based approach provides a solid foundation for scalable, maintainable operator deployment automation. 