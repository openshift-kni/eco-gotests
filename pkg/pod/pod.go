package pod

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	multus "gopkg.in/k8snetworkplumbingwg/multus-cni.v3/pkg/types"

	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/utils/pointer"

	"github.com/golang/glog"

	"github.com/openshift-kni/eco-gotests/pkg/clients"
)

// Builder provides a struct for pod object from the cluster and a pod definition.
type Builder struct {
	// Pod definition, used to create the pod object.
	Definition *v1.Pod
	// Created pod object.
	Object *v1.Pod
	// Used to store latest error message upon defining or mutating pod definition.
	errorMsg string
	// api client to interact with the cluster.
	apiClient *clients.Settings
}

// NewBuilder creates a new instance of Builder.
func NewBuilder(apiClient *clients.Settings, name, nsname, image string) *Builder {
	glog.V(100).Infof(
		"Initializing new pod structure with the following params: "+
			"name: %s, namespace: %s, image: %s",
		name, nsname, image)

	builder := &Builder{
		apiClient:  apiClient,
		Definition: getDefinition(name, nsname),
	}

	if name == "" {
		glog.V(100).Infof("The name of the pod is empty")

		builder.errorMsg = "pod's name is empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the pod is empty")

		builder.errorMsg = "namespace's name is empty"
	}

	if image == "" {
		glog.V(100).Infof("The image of the pod is empty")

		builder.errorMsg = "pod's image is empty"
	}

	defaultContainer, err := NewContainerBuilder("test", image, []string{"/bin/bash", "-c", "sleep INF"}).GetContainerCfg()

	if err != nil {
		glog.V(100).Infof("Failed to define the default container settings")

		builder.errorMsg = err.Error()
	}

	builder.Definition.Spec.Containers = append(builder.Definition.Spec.Containers, *defaultContainer)

	return builder
}

// Pull loads an existing pod into the Builder struct.
func Pull(apiClient *clients.Settings, name, nsname string) (*Builder, error) {
	glog.V(100).Infof("Pulling existing pod name: %s namespace:%s", name, nsname)

	builder := Builder{
		apiClient: apiClient,
		Definition: &v1.Pod{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the pod is empty")

		builder.errorMsg = "pod 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the pod is empty")

		builder.errorMsg = "pod 'namespace' cannot be empty"
	}

	if builder.errorMsg != "" {
		return nil, fmt.Errorf("faield to pull pod object due to the following error: %s", builder.errorMsg)
	}

	if !builder.Exists() {
		glog.V(100).Infof("Failed to pull pod object %s from namespace %s. Object doesn't exist",
			name, nsname)

		return nil, fmt.Errorf("pod object %s doesn't exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// DefineOnNode adds nodeName to the pod's definition.
func (builder *Builder) DefineOnNode(nodeName string) *Builder {
	glog.V(100).Infof("Adding nodeName %s to the definition of pod %s in namespace %s",
		nodeName, builder.Definition.Name, builder.Definition.Namespace)

	if builder.Definition == nil {
		glog.V(100).Infof("The pod is undefined")

		builder.errorMsg = "can not define pod on specific node because basic definition is empty"
	}

	if builder.Object != nil {
		glog.V(100).Infof("The pod is already running on node %s", builder.Object.Spec.NodeName)

		builder.errorMsg = fmt.Sprintf(
			"can not redefine running pod. pod already running on node %s", builder.Object.Spec.NodeName)
	}

	if nodeName == "" {
		glog.V(100).Infof("The node name is empty")

		builder.errorMsg = "can not define pod on empty node"
	}

	if builder.errorMsg == "" {
		builder.Definition.Spec.NodeName = nodeName
	}

	return builder
}

// Create makes a pod according to the pod definition and stores the created object in the pod builder.
func (builder *Builder) Create() (*Builder, error) {
	glog.V(100).Infof("Creating pod %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	if !builder.Exists() {
		builder.Object, err = builder.apiClient.Pods(builder.Definition.Namespace).Create(
			context.TODO(), builder.Definition, metaV1.CreateOptions{})
	}

	return builder, err
}

// Delete removes the pod object and resets the builder object.
func (builder *Builder) Delete() (*Builder, error) {
	glog.V(100).Infof("Deleting pod %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return builder, fmt.Errorf("pod cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Pods(builder.Definition.Namespace).Delete(
		context.TODO(), builder.Object.Name, metaV1.DeleteOptions{})

	if err != nil {
		return builder, fmt.Errorf("can not delete pod: %w", err)
	}

	builder.Object = nil

	return builder, nil
}

// DeleteAndWait deletes the pod object and waits until the pod is deleted.
func (builder *Builder) DeleteAndWait(timeout time.Duration) (*Builder, error) {
	glog.V(100).Infof("Deleting pod %s in namespace %s and waiting for the defined period until it's removed",
		builder.Definition.Name, builder.Definition.Namespace)

	builder, err := builder.Delete()
	if err != nil {
		return builder, err
	}

	err = builder.WaitUntilDeleted(timeout)

	if err != nil {
		return builder, err
	}

	return builder, nil
}

// CreateAndWaitUntilRunning creates the pod object and waits until the pod is running.
func (builder *Builder) CreateAndWaitUntilRunning(timeout time.Duration) (*Builder, error) {
	glog.V(100).Infof("Creating pod %s in namespace %s and waiting for the defined period until it's ready",
		builder.Definition.Name, builder.Definition.Namespace)

	builder, err := builder.Create()
	if err != nil {
		return builder, err
	}

	err = builder.WaitUntilRunning(timeout)

	if err != nil {
		return builder, err
	}

	return builder, nil
}

// WaitUntilRunning waits for the duration of the defined timeout or until the pod is running.
func (builder *Builder) WaitUntilRunning(timeout time.Duration) error {
	glog.V(100).Infof("Waiting for the defined period until pod %s in namespace %s is running",
		builder.Definition.Name, builder.Definition.Namespace)

	return builder.WaitUntilInStatus(v1.PodRunning, timeout)
}

// WaitUntilInStatus waits for the duration of the defined timeout or until the pod gets to a specific status.
func (builder *Builder) WaitUntilInStatus(status v1.PodPhase, timeout time.Duration) error {
	glog.V(100).Infof("Waiting for the defined period until pod %s in namespace %s has status %v",
		builder.Definition.Name, builder.Definition.Namespace, status)

	if builder.errorMsg != "" {
		return fmt.Errorf(builder.errorMsg)
	}

	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		updatePod, err := builder.apiClient.Pods(builder.Object.Namespace).Get(
			context.Background(), builder.Object.Name, metaV1.GetOptions{})
		if err != nil {
			return false, nil
		}

		return updatePod.Status.Phase == status, nil
	})
}

// WaitUntilDeleted waits for the duration of the defined timeout or until the pod is deleted.
func (builder *Builder) WaitUntilDeleted(timeout time.Duration) error {
	glog.V(100).Infof("Waiting for the defined period until pod %s in namespace %s is deleted",
		builder.Definition.Name, builder.Definition.Namespace)

	err := wait.Poll(time.Second, timeout, func() (bool, error) {
		_, err := builder.apiClient.Pods(builder.Definition.Namespace).Get(
			context.Background(), builder.Definition.Name, metaV1.GetOptions{})
		if err == nil {
			glog.V(100).Infof("pod %s/%s still present", builder.Definition.Namespace, builder.Definition.Name)

			return false, nil
		}
		if k8serrors.IsNotFound(err) {
			glog.V(100).Infof("pod %s/%s is gone", builder.Definition.Namespace, builder.Definition.Name)

			return true, nil
		}
		glog.V(100).Infof("failed to get pod %s/%s: %v", builder.Definition.Namespace, builder.Definition.Name, err)

		return false, err
	})

	return err
}

// ExecCommand runs command in the pod and returns the buffer output.
func (builder *Builder) ExecCommand(command []string, containerName ...string) (bytes.Buffer, error) {
	glog.V(100).Infof("Execute command %v in the pod",
		command)

	var (
		buffer bytes.Buffer
		cName  string
	)

	if len(containerName) > 0 {
		cName = containerName[0]
	} else {
		cName = builder.Definition.Spec.Containers[0].Name
	}

	req := builder.apiClient.CoreV1Interface.RESTClient().
		Post().
		Namespace(builder.Object.Namespace).
		Resource("pods").
		Name(builder.Object.Name).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Container: cName,
			Command:   command,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(builder.apiClient.Config, "POST", req.URL())

	if err != nil {
		return buffer, err
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: &buffer,
		Stderr: os.Stderr,
		Tty:    true,
	})

	if err != nil {
		return buffer, err
	}

	return buffer, nil
}

// Exists checks whether the given namespace exists.
func (builder *Builder) Exists() bool {
	glog.V(100).Infof("Checking if pod %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.apiClient.Pods(builder.Definition.Namespace).Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// RedefineDefaultCMD redefines default command in pod's definition.
func (builder *Builder) RedefineDefaultCMD(command []string) *Builder {
	glog.V(100).Infof("Redefining default pod's container cmd with the new %v", command)

	builder.isMutationAllowed("cmd")

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.Containers[0].Command = command

	return builder
}

// WithRestartPolicy applies restart policy to pod's definition.
func (builder *Builder) WithRestartPolicy(restartPolicy v1.RestartPolicy) *Builder {
	glog.V(100).Infof("Redefining pod's RestartPolicy to %v", restartPolicy)

	builder.isMutationAllowed("RestartPolicy")

	if restartPolicy == "" {
		glog.V(100).Infof(
			"Failed to set RestartPolicy on pod %s in namespace %s. RestartPolicy can not be empty",
			builder.Definition.Name, builder.Definition.Namespace)

		builder.errorMsg = "can not define pod with empty restart policy"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.RestartPolicy = restartPolicy

	return builder
}

// WithTolerationToMaster sets toleration policy which allows pod to be running on master node.
func (builder *Builder) WithTolerationToMaster() *Builder {
	glog.V(100).Infof("Redefining pod's %s with toleration to master node", builder.Definition.Name)

	builder.isMutationAllowed("toleration to master node")

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.Tolerations = []v1.Toleration{
		{
			Key:    "node-role.kubernetes.io/master",
			Effect: "NoSchedule",
		},
	}

	return builder
}

// WithPrivilegedFlag sets privileged flag on all containers.
func (builder *Builder) WithPrivilegedFlag() *Builder {
	glog.V(100).Infof("Applying privileged flag to all pod's: %s containers", builder.Definition.Name)

	builder.isMutationAllowed("privileged container flag")

	if builder.errorMsg != "" {
		return builder
	}

	for idx := range builder.Definition.Spec.Containers {
		builder.Definition.Spec.Containers[idx].SecurityContext = &v1.SecurityContext{}
		trueFlag := true
		builder.Definition.Spec.Containers[idx].SecurityContext.Privileged = &trueFlag
	}

	return builder
}

// WithLocalVolume attaches given volume to all pod's containers.
func (builder *Builder) WithLocalVolume(volumeName, mountPath string) *Builder {
	glog.V(100).Infof("Configuring volume %s for all pod's: %s containers. MountPath %s",
		volumeName, builder.Definition.Name, mountPath)

	builder.isMutationAllowed("LocalVolume")

	if volumeName == "" {
		glog.V(100).Infof("The 'volumeName' of the pod is empty")

		builder.errorMsg = "'volumeName' parameter is empty"
	}

	if mountPath == "" {
		glog.V(100).Infof("The 'mountPath' of the pod is empty")

		builder.errorMsg = "'mountPath' parameter is empty"
	}

	mountConfig := v1.VolumeMount{Name: volumeName, MountPath: mountPath, ReadOnly: false}

	builder.isMountAlreadyInUseInPod(mountConfig)

	if builder.errorMsg != "" {
		return builder
	}

	for index := range builder.Definition.Spec.Containers {
		builder.Definition.Spec.Containers[index].VolumeMounts = append(
			builder.Definition.Spec.Containers[index].VolumeMounts, mountConfig)
	}

	if len(builder.Definition.Spec.InitContainers) > 0 {
		for index := range builder.Definition.Spec.InitContainers {
			builder.Definition.Spec.InitContainers[index].VolumeMounts = append(
				builder.Definition.Spec.InitContainers[index].VolumeMounts, mountConfig)
		}
	}

	builder.Definition.Spec.Volumes = append(builder.Definition.Spec.Volumes,
		v1.Volume{Name: mountConfig.Name, VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: mountConfig.Name,
				},
			},
		}})

	return builder
}

// WithAdditionalContainer appends additional container to pod.
func (builder *Builder) WithAdditionalContainer(container *v1.Container) *Builder {
	glog.V(100).Infof("Adding new container %v to pod %s", container, builder.Definition.Name)
	builder.isMutationAllowed("additional container")

	if container == nil {
		builder.errorMsg = "'container' parameter cannot be empty"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.Containers = append(builder.Definition.Spec.Containers, *container)

	return builder
}

// WithSecondaryNetwork applies Multus secondary network on pod definition.
func (builder *Builder) WithSecondaryNetwork(network []*multus.NetworkSelectionElement) *Builder {
	glog.V(100).Infof("Applying secondary network %v to pod %s", network, builder.Definition.Name)

	builder.isMutationAllowed("secondary network")

	if builder.errorMsg != "" {
		return builder
	}

	netAnnotation, err := json.Marshal(network)

	if err != nil {
		builder.errorMsg = fmt.Sprintf("error to unmarshal network annotation due to: %s", err.Error())
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Annotations = map[string]string{"k8s.v1.cni.cncf.io/networks": string(netAnnotation)}

	return builder
}

// PullImage pulls image for given pod's container and removes it.
func (builder *Builder) PullImage(timeout time.Duration, testCmd []string) error {
	glog.V(100).Infof(
		"Pulling container image %s to node: %s", builder.Definition.Spec.Containers[0].Image,
		builder.Definition.Spec.NodeName)

	builder.WithRestartPolicy(v1.RestartPolicyNever)
	builder.RedefineDefaultCMD(testCmd)
	_, err := builder.Create()

	if err != nil {
		glog.V(100).Infof(
			"Failed to create pod %s in namespace %s and pull image %s to node: %s",
			builder.Definition.Name, builder.Definition.Namespace, builder.Definition.Spec.Containers[0].Image,
			builder.Definition.Spec.NodeName)

		return err
	}

	err = builder.WaitUntilInStatus(v1.PodSucceeded, timeout)

	if err != nil {
		glog.V(100).Infof(
			"Pod status timeout %s. Pod is not in status Succeeded in namespace %s. "+
				"Fail to confirm that image %s was pulled to node: %s",
			builder.Definition.Name, builder.Definition.Namespace, builder.Definition.Spec.Containers[0].Image,
			builder.Definition.Spec.NodeName)

		_, err = builder.Delete()

		if err != nil {
			glog.V(100).Infof(
				"Failed to remove pod %s in namespace %s from node: %s",
				builder.Definition.Name, builder.Definition.Namespace, builder.Definition.Spec.NodeName)

			return err
		}

		return err
	}

	_, err = builder.Delete()

	return err
}

func getDefinition(name, nsName string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: nsName},
		Spec: v1.PodSpec{
			TerminationGracePeriodSeconds: pointer.Int64(0),
		},
	}
}

func (builder *Builder) isMutationAllowed(configToMutate string) {
	if builder.Definition == nil {
		glog.V(100).Infof(
			"Failed to redefine pod's %s because basic pod %s definition is empty in namespace %s",
			builder.Definition.Name, configToMutate, builder.Definition.Namespace)

		builder.errorMsg = fmt.Sprintf("can not define pod with %s because basic pod definition is empty",
			configToMutate)
	}

	if builder.Object != nil {
		glog.V(100).Infof(
			"Failed to redefine %s for running pod %s in namespace %s",
			builder.Definition.Name, configToMutate, builder.Definition.Namespace)

		builder.errorMsg = fmt.Sprintf(
			"can not redefine running pod. pod already running on node %s", builder.Object.Spec.NodeName)
	}
}

func (builder *Builder) isMountAlreadyInUseInPod(newMount v1.VolumeMount) {
	for index := range builder.Definition.Spec.Containers {
		if builder.Definition.Spec.Containers[index].VolumeMounts != nil {
			if isMountInUse(builder.Definition.Spec.Containers[index].VolumeMounts, newMount) {
				builder.errorMsg = fmt.Sprintf("given mount %v already mounted to pod's container %s",
					newMount.Name, builder.Definition.Spec.Containers[index].Name)
			}
		}
	}
}

func isMountInUse(containerMounts []v1.VolumeMount, newMount v1.VolumeMount) bool {
	for _, containerMount := range containerMounts {
		if containerMount.Name == newMount.Name && containerMount.MountPath == newMount.MountPath {
			return true
		}
	}

	return false
}
