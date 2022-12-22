package pod

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

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
func NewBuilder(apiClient *clients.Settings, name, nsName, image string) *Builder {
	builder := &Builder{
		apiClient:  apiClient,
		Definition: getDefinition(name, nsName, image),
	}

	if name == "" {
		builder.errorMsg = "pod's name is empty"
	}

	if nsName == "" {
		builder.errorMsg = "namespace's name is empty"
	}

	if image == "" {
		builder.errorMsg = "pod's image is empty"
	}

	return builder
}

// DefineOnNode adds node name in the pod's definition.
func (builder *Builder) DefineOnNode(nodeName string) *Builder {
	if builder.Definition == nil {
		builder.errorMsg = "can not define pod on specific node because basic definition is empty"
	}

	if builder.Object != nil {
		builder.errorMsg = fmt.Sprintf(
			"can not redefine running pod. pod already running on node %s", builder.Object.Spec.NodeName)
	}

	if nodeName == "" {
		builder.errorMsg = "can not define pod on empty node"
	}

	if builder.errorMsg == "" {
		builder.Definition.Spec.NodeName = nodeName
	}

	return builder
}

// Create makes a pod according to the pod definition and stores the created object in the pod builder.
func (builder *Builder) Create() (*Builder, error) {
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
	return builder.WaitUntilInStatus(v1.PodRunning, timeout)
}

// WaitUntilInStatus waits for the duration of the defined timeout or until the pod gets to a specific status.
func (builder *Builder) WaitUntilInStatus(status v1.PodPhase, timeout time.Duration) error {
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
	var err error
	builder.Object, err = builder.apiClient.Pods(builder.Definition.Namespace).Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

func getDefinition(name, nsName, image string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			GenerateName: name,
			Namespace:    nsName},
		Spec: v1.PodSpec{
			TerminationGracePeriodSeconds: pointer.Int64Ptr(0),
			Containers: []v1.Container{
				{
					Name:    "test",
					Image:   image,
					Command: []string{"/bin/bash", "-c", "sleep INF"},
				},
			},
		},
	}
}
