package nto //nolint:misspell

import (
	"context"
	"fmt"

	"k8s.io/utils/strings/slices"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	v2 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/performanceprofile/v2"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Builder provides a struct for PerformanceProfile object from the cluster and a PerformanceProfile definition.
type Builder struct {
	// PerformanceProfile definition, used to create the PerformanceProfile object.
	Definition *v2.PerformanceProfile
	// Created PerformanceProfile object.
	Object *v2.PerformanceProfile
	// Used to store latest error message upon defining or mutating PerformanceProfile definition.
	errorMsg string
	// api client to interact with the cluster.
	apiClient *clients.Settings
}

// NewBuilder creates a new instance of Builder.
func NewBuilder(
	apiClient *clients.Settings, name, cpuIsolated, cpuReserved string, nodeSelector map[string]string) *Builder {
	glog.V(100).Infof(
		"Initializing new PerformanceProfile structure with the following params: "+
			"name: %s, cpu isolated: %s, cpu reserved %s, nodeSelector %v", name, cpuIsolated, cpuReserved, nodeSelector)

	isolatedCPUSet := v2.CPUSet(cpuIsolated)
	reservedCPUSet := v2.CPUSet(cpuReserved)

	builder := &Builder{
		apiClient: apiClient,
		Definition: &v2.PerformanceProfile{
			ObjectMeta: metaV1.ObjectMeta{
				Name: name,
			},
			Spec: v2.PerformanceProfileSpec{
				CPU: &v2.CPU{
					Isolated: &isolatedCPUSet,
					Reserved: &reservedCPUSet,
				},
				NodeSelector: nodeSelector,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the PerformanceProfile is empty")

		builder.errorMsg = "PerformanceProfile's name is empty"
	}

	if cpuIsolated == "" {
		glog.V(100).Infof("Isolated CPU of the PerformanceProfile is empty")

		builder.errorMsg = "PerformanceProfile's 'cpuIsolated' is empty"
	}

	if cpuReserved == "" {
		glog.V(100).Infof("Reserved CPU of the PerformanceProfile is empty")

		builder.errorMsg = "PerformanceProfile's 'cpuReserved' is empty"
	}

	if len(nodeSelector) == 0 {
		glog.V(100).Infof("NodeSelector of the PerformanceProfile is empty")

		builder.errorMsg = "PerformanceProfile's 'nodeSelector' is empty"
	}

	return builder
}

// Pull pulls existing PerformanceProfile from cluster.
func Pull(apiClient *clients.Settings, name string) (*Builder, error) {
	glog.V(100).Infof("Pulling existing PerformanceProfile name %s from cluster", name)

	builder := Builder{
		apiClient: apiClient,
		Definition: &v2.PerformanceProfile{
			ObjectMeta: metaV1.ObjectMeta{
				Name: name,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the PerformanceProfile is empty")

		builder.errorMsg = "PerformanceProfile 'name' cannot be empty"
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("PerformanceProfile object %s doesn't exist", name)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// WithHugePages defines the HugePages in the PerformanceProfile. hugePageSize allowed values are 2M, 1G.
func (builder *Builder) WithHugePages(hugePageSize string, hugePages []v2.HugePage) *Builder {
	glog.V(100).Infof("Adding hugePages to PerformanceProfile %s, size %s, hugePages %v",
		builder.Definition.Name, hugePageSize, hugePages)

	if valid, _ := builder.validate(); !valid {
		return builder
	}

	allowedHugePageSize := []string{"2M", "1G"}
	if !slices.Contains(allowedHugePageSize, hugePageSize) {
		glog.V(100).Infof("'hugePageSize' has invalid parameter %s. Allowed parameters %v",
			hugePageSize, allowedHugePageSize)

		builder.errorMsg = fmt.Sprintf("'hugePageSize' argument is not in allowed list %v", allowedHugePageSize)
	}

	if len(hugePages) == 0 {
		glog.V(100).Infof("'hugePages' argument cannot be empty")

		builder.errorMsg = "'hugePageSize' argument cannot be empty"
	}

	if builder.errorMsg != "" {
		return builder
	}

	pageSize := v2.HugePageSize(hugePageSize)

	if builder.Definition.Spec.HugePages != nil {
		builder.Definition.Spec.HugePages.DefaultHugePagesSize = &pageSize
		builder.Definition.Spec.HugePages.Pages = hugePages

		return builder
	}

	builder.Definition.Spec.HugePages = &v2.HugePages{
		DefaultHugePagesSize: &pageSize,
		Pages:                hugePages,
	}

	return builder
}

// WithMachineConfigPoolSelector defines the MachineConfigPoolSelector in the PerformanceProfile.
func (builder *Builder) WithMachineConfigPoolSelector(machineConfigPoolSelector map[string]string) *Builder {
	glog.V(100).Infof("Adding MachineConfigPoolSelector %v to PerformanceProfile %s",
		machineConfigPoolSelector, builder.Definition.Name)

	if valid, _ := builder.validate(); !valid {
		return builder
	}

	if len(machineConfigPoolSelector) == 0 {
		glog.V(100).Infof("'machineConfigPoolSelector' argument cannot be empty")

		builder.errorMsg = "'machineConfigPoolSelector' argument cannot be empty"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.MachineConfigPoolSelector = machineConfigPoolSelector

	return builder
}

// WithNumaTopology defines the NumaTopologyPolicy in the PerformanceProfile.
func (builder *Builder) WithNumaTopology(topologyPolicy string) *Builder {
	glog.V(100).Infof("Adding NumaTopologyPolicy %s to PerformanceProfile %s",
		topologyPolicy, builder.Definition.Name)

	if valid, _ := builder.validate(); !valid {
		return builder
	}

	allowedTopologyPolicies := []string{"best-effort", "restricted", "single-numa-node"}
	if !slices.Contains(allowedTopologyPolicies, topologyPolicy) {
		glog.V(100).Infof("'allowedTopologyPolicies' has invalid parameter %s. Allowed parameters %v",
			topologyPolicy, allowedTopologyPolicies)

		builder.errorMsg = fmt.Sprintf("'allowedTopologyPolicies' argument is not in allowed list %v",
			allowedTopologyPolicies)
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.NUMA.TopologyPolicy = &topologyPolicy

	return builder
}

// WithRTKernel defines the Real Time Kernel in the PerformanceProfile.
func (builder *Builder) WithRTKernel() *Builder {
	glog.V(100).Infof("Adding RTKernel flag to PerformanceProfile %s", builder.Definition.Name)

	if valid, _ := builder.validate(); !valid {
		return builder
	}

	trueFlag := true
	builder.Definition.Spec.RealTimeKernel = &v2.RealTimeKernel{Enabled: &trueFlag}

	return builder
}

// WithWorkloadHints defines the Workload Hints in the PerformanceProfile.
func (builder *Builder) WithWorkloadHints(rtHint, perPodPowerMgmtHint, highPowerHint bool) *Builder {
	glog.V(100).Infof(
		"Adding WorkloadHints flags: RealTime=%t, PerPodPowerManagement=%t, HighPowerConsumption=%t to PerformanceProfile %s",
		rtHint, perPodPowerMgmtHint, highPowerHint, builder.Definition.Name)

	if valid, _ := builder.validate(); !valid {
		return builder
	}

	if builder.Definition.Spec.WorkloadHints == nil {
		builder.Definition.Spec.WorkloadHints = &v2.WorkloadHints{
			RealTime:              &rtHint,
			PerPodPowerManagement: &perPodPowerMgmtHint,
			HighPowerConsumption:  &highPowerHint,
		}

		return builder
	}

	builder.Definition.Spec.WorkloadHints.RealTime = &rtHint
	builder.Definition.Spec.WorkloadHints.PerPodPowerManagement = &perPodPowerMgmtHint
	builder.Definition.Spec.WorkloadHints.HighPowerConsumption = &highPowerHint

	return builder
}

// Create the PerformanceProfile in the cluster and store the created object in Object.
func (builder *Builder) Create() (*Builder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating PerformanceProfile %s ", builder.Definition.Name)

	var err error
	if !builder.Exists() {
		err = builder.apiClient.Create(context.TODO(), builder.Definition)

		if err != nil {
			return nil, err
		}

		builder.Object, err = builder.Get()
	}

	return builder, err
}

// Exists checks whether the given PerformanceProfile exists.
func (builder *Builder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if PerformanceProfile %s exists", builder.Definition.Name)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// Get fetches the defined PerformanceProfile from the cluster.
func (builder *Builder) Get() (*v2.PerformanceProfile, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Getting PerformanceProfile %s", builder.Definition.Name)

	module := &v2.PerformanceProfile{}

	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name: builder.Definition.Name,
	}, module)

	if err != nil {
		return nil, err
	}

	return module, err
}

// Delete removes the PerformanceProfile.
func (builder *Builder) Delete() (*Builder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Deleting PerformanceProfile %s", builder.Definition.Name)

	if !builder.Exists() {
		return builder, fmt.Errorf("PerformanceProfile cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return builder, err
	}

	builder.Object = nil

	return builder, err
}

// Update renovates the existing PerformanceProfile object with the PerformanceProfile definition in builder.
func (builder *Builder) Update(force bool) (*Builder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating the PerformanceProfile object: %s", builder.Definition.Name)

	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	if err != nil {
		if force {
			glog.V(100).Infof(
				"Failed to update the PerformanceProfile object %s. "+
					"Note: Force flag set, executed delete/create methods instead", builder.Definition.Name)

			builder, err := builder.Delete()

			if err != nil {
				glog.V(100).Infof(
					"Failed to update the PerformanceProfile object %s, "+
						"due to error in delete function", builder.Definition.Name)

				return nil, err
			}

			return builder.Create()
		}
	}

	if err == nil {
		builder.Object = builder.Definition
	}

	return builder, err
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *Builder) validate() (bool, error) {
	resourceCRD := "PerformanceProfile"

	if builder == nil {
		glog.V(100).Infof("The %s builder is uninitialized", resourceCRD)

		return false, fmt.Errorf("error: received nil %s builder", resourceCRD)
	}

	if builder.Definition == nil {
		glog.V(100).Infof("The %s is undefined", resourceCRD)

		builder.errorMsg = msg.UndefinedCrdObjectErrString(resourceCRD)
	}

	if builder.apiClient == nil {
		glog.V(100).Infof("The %s builder apiclient is nil", resourceCRD)

		builder.errorMsg = fmt.Sprintf("%s builder cannot have nil apiClient", resourceCRD)
	}

	if builder.errorMsg != "" {
		glog.V(100).Infof("The %s builder has error message: %s", resourceCRD, builder.errorMsg)

		return false, fmt.Errorf(builder.errorMsg)
	}

	return true, nil
}
