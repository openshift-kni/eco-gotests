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
- **`amd-deployer.go`**: AMD GPU operator

### Factory Pattern (`factory.go`)
- **`DeployerFactory`**: Centralized deployer creation with defaults
- **`DeployerSet`**: Container for managing multiple deployers
- **`DeployerType`**: Type-safe deployer identification

## Factory Pattern Usage (Recommended)

The **DeployerFactory** provides the easiest way to create and manage deployers with sensible defaults.

### Quick Start Example

```go
// Create factory with Kubernetes API client
factory := deploy.NewDeployerFactory(APIClient)

// Create NFD deployer with default configuration
nfdDeployer, err := factory.CreateNFDDeployer(nil)
if err != nil {
    return fmt.Errorf("failed to create NFD deployer: %w", err)
}

// Deploy the NFD operator
err = nfdDeployer.Deploy()
if err != nil {
    return fmt.Errorf("failed to deploy NFD operator: %w", err)
}

// Wait for operator to be ready
ready, err := nfdDeployer.IsReady(5 * time.Minute)
if err != nil || !ready {
    return fmt.Errorf("NFD operator not ready: %w", err)
}

// Deploy NodeFeatureDiscovery custom resource
nfdConfig := deploy.NFDConfig{
    EnableTopology: true,
    Image:          "registry.redhat.io/openshift4/ose-node-feature-discovery:latest",
}

err = nfdDeployer.DeployCustomResource("nfd-instance", nfdConfig)
if err != nil {
    return fmt.Errorf("failed to deploy NFD custom resource: %w", err)
}

// Wait for custom resource to be ready
crReady, err := nfdDeployer.IsCustomResourceReady("nfd-instance", 3*time.Minute)
if err != nil || !crReady {
    return fmt.Errorf("NFD custom resource not ready: %w", err)
}

// Cleanup when done
defer func() {
    nfdDeployer.DeleteCustomResource("nfd-instance")
    nfdDeployer.Undeploy()
}()
```

### Create Multiple Deployers

```go
factory := deploy.NewDeployerFactory(APIClient)

// Create all deployers with defaults
deployers := factory.CreateAllDeployers()

// Deploy all operators
err := deployers.DeployAll()
if err != nil {
    return fmt.Errorf("failed to deploy all operators: %w", err)
}

// Wait for all to be ready
// (individual readiness checks as needed)

// Cleanup all
defer deployers.UndeployAll()
```

### Custom Configuration

```go
factory := deploy.NewDeployerFactory(APIClient)

// Override NFD configuration
nfdConfig := &deploy.OperatorConfig{
    Namespace: "custom-nfd-namespace",
    Channel:   "alpha",
}

// Create NFD deployer with custom config
nfdDeployer, err := factory.CreateNFDDeployer(nfdConfig)
if err != nil {
    return fmt.Errorf("failed to create NFD deployer: %w", err)
}
```

## NFD Custom Resource Configuration

### NFDConfig Structure
```go
type NFDConfig struct {
    EnableTopology bool   `json:"enableTopology,omitempty"`
    Image          string `json:"image,omitempty"`
}
```

### NFD CR Deployment Examples

#### Basic NFD Custom Resource
```go
// Simple configuration
nfdConfig := deploy.NFDConfig{
    EnableTopology: true,
}

err = nfdDeployer.DeployCustomResource("nfd-basic", nfdConfig)
```

#### NFD with Custom Image
```go
// Custom image configuration
nfdConfig := deploy.NFDConfig{
    EnableTopology: true,
    Image:          "quay.io/openshift/origin-node-feature-discovery:latest",
}

err = nfdDeployer.DeployCustomResource("nfd-custom", nfdConfig)
```

#### Multiple NFD Instances
```go
// Deploy multiple NFD instances for different configurations
configs := map[string]deploy.NFDConfig{
    "nfd-prod": {
        EnableTopology: true,
        Image:          "registry.redhat.io/openshift4/ose-node-feature-discovery:latest",
    },
    "nfd-test": {
        EnableTopology: false,
        Image:          "quay.io/openshift/origin-node-feature-discovery:latest",
    },
}

for name, config := range configs {
    err := nfdDeployer.DeployCustomResource(name, config)
    if err != nil {
        log.Errorf("Failed to deploy %s: %v", name, err)
        continue
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

### ✅ **Factory Pattern Benefits**
- Sensible defaults out of the box
- Configuration override capabilities
- Batch operations support
- Type-safe deployer creation

### ✅ **Testability**
- Easy mocking via interfaces
- Isolated testing of components
- Consistent patterns for testing

### ✅ **Maintainability**
- Clear code organization
- Reduced duplication
- Type-safe operations

## Default Configurations

The factory provides default configurations for each operator:

### NFD Defaults
- **Namespace**: `openshift-nfd`
- **Channel**: `stable`
- **Catalog**: `certified-operators`
- **Package**: `nfd`

### KMM Defaults
- **Namespace**: `openshift-kmm`
- **Channel**: `stable`
- **Catalog**: `certified-operators`
- **Package**: `kernel-module-management`

### AMD GPU Defaults
- **Namespace**: `openshift-operators` (global)
- **Channel**: `alpha`
- **Catalog**: `certified-operators`
- **Package**: `amd-gpu-operator`

## Adding New Operators

To add a new operator:

1. **Create deployer file**: `new-operator-deployer.go`
2. **Implement interfaces**: `OperatorDeployer` and `CustomResourceDeployer`
3. **Add to factory**: Update `factory.go` with new deployer type and methods
4. **Define config struct**: Operator-specific configuration (if needed)
5. **Create tests**: `new-operator-deployer_test.go`

This interface-based approach with factory pattern provides a solid foundation for scalable, maintainable operator deployment automation. 