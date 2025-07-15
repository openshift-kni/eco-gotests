package deploy

import (
	"fmt"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
)

// ExampleUsage demonstrates how to use the new interface-based deployers
func ExampleUsage() {
	// This is an example function showing how to use the new deployers
	// In real usage, you would get the API client from your test setup

	var apiClient *clients.Settings // This would be your real API client

	// Example 1: Deploy NFD operator
	fmt.Println("=== NFD Deployment Example ===")

	// Create NFD operator configuration
	nfdConfig := NewOperatorConfig(
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

	// Create NFD deployer
	nfdDeployer := NewNFDDeployer(nfdConfig)

	// Deploy the operator
	if err := nfdDeployer.Deploy(); err != nil {
		fmt.Printf("Failed to deploy NFD operator: %v\n", err)
		return
	}

	// Wait for operator to be ready
	ready, err := nfdDeployer.IsReady(5 * time.Minute)
	if err != nil {
		fmt.Printf("Error checking NFD readiness: %v\n", err)
		return
	}

	if !ready {
		fmt.Println("NFD operator is not ready yet")
		return
	}

	// Deploy NFD custom resource
	nfdCRConfig := NFDConfig{
		EnableTopology: true,
		Image:          "registry.redhat.io/openshift4/ose-node-feature-discovery:latest",
	}

	if err := nfdDeployer.DeployCustomResource("nfd-instance", nfdCRConfig); err != nil {
		fmt.Printf("Failed to deploy NFD CR: %v\n", err)
		return
	}

	fmt.Println("NFD deployment completed successfully!")

	// Example 2: Deploy KMM operator
	fmt.Println("\n=== KMM Deployment Example ===")

	// Create KMM operator configuration
	kmmConfig := NewOperatorConfig(
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

	// Create KMM deployer
	kmmDeployer := NewKMMDeployer(kmmConfig)

	// Deploy the operator
	if err := kmmDeployer.Deploy(); err != nil {
		fmt.Printf("Failed to deploy KMM operator: %v\n", err)
		return
	}

	// Wait for operator to be ready
	ready, err = kmmDeployer.IsReady(5 * time.Minute)
	if err != nil {
		fmt.Printf("Error checking KMM readiness: %v\n", err)
		return
	}

	if !ready {
		fmt.Println("KMM operator is not ready yet")
		return
	}

	// Deploy a simple KMM module
	nodeSelector := map[string]string{
		"node-role.kubernetes.io/worker": "",
	}

	if err := kmmDeployer.DeploySimpleModule(
		"example-module",
		".*",
		"example.com/kernel-module:latest",
		nodeSelector,
	); err != nil {
		fmt.Printf("Failed to deploy KMM module: %v\n", err)
		return
	}

	// Or deploy with custom configuration
	kmmCRConfig := KMMConfig{
		ModuleName:     "custom-module",
		KernelMapping:  "5.14.*",
		ContainerImage: "example.com/custom-module:v2",
		NodeSelector:   nodeSelector,
	}

	if err := kmmDeployer.DeployCustomResource("custom-module", kmmCRConfig); err != nil {
		fmt.Printf("Failed to deploy custom KMM module: %v\n", err)
		return
	}

	fmt.Println("KMM deployment completed successfully!")

	// Example 3: Using the interface for polymorphic behavior
	fmt.Println("\n=== Polymorphic Usage Example ===")

	deployers := []OperatorDeployer{nfdDeployer, kmmDeployer}

	for i, deployer := range deployers {
		fmt.Printf("Deployer %d: %s in namespace %s\n",
			i+1, deployer.GetOperatorName(), deployer.GetNamespace())

		// You can call common interface methods
		ready, err := deployer.IsReady(30 * time.Second)
		if err != nil {
			fmt.Printf("Error checking readiness: %v\n", err)
			continue
		}

		fmt.Printf("Operator ready: %t\n", ready)
	}
}

// ExampleCleanup demonstrates how to clean up deployed resources
func ExampleCleanup() {
	var apiClient *clients.Settings // This would be your real API client

	fmt.Println("=== Cleanup Example ===")

	// Create deployers with the same configs used for deployment
	nfdConfig := NewOperatorConfig(
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

	kmmConfig := NewOperatorConfig(
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

	nfdDeployer := NewNFDDeployer(nfdConfig)
	kmmDeployer := NewKMMDeployer(kmmConfig)

	// Clean up custom resources first
	if err := nfdDeployer.DeleteCustomResource("nfd-instance"); err != nil {
		fmt.Printf("Failed to delete NFD CR: %v\n", err)
	}

	if err := kmmDeployer.DeleteCustomResource("example-module"); err != nil {
		fmt.Printf("Failed to delete KMM module: %v\n", err)
	}

	if err := kmmDeployer.DeleteCustomResource("custom-module"); err != nil {
		fmt.Printf("Failed to delete custom KMM module: %v\n", err)
	}

	// Clean up operators
	deployers := []OperatorDeployer{nfdDeployer, kmmDeployer}

	for _, deployer := range deployers {
		fmt.Printf("Cleaning up %s...\n", deployer.GetOperatorName())

		if err := deployer.Undeploy(); err != nil {
			fmt.Printf("Failed to undeploy %s: %v\n", deployer.GetOperatorName(), err)
		} else {
			fmt.Printf("Successfully cleaned up %s\n", deployer.GetOperatorName())
		}
	}
}
