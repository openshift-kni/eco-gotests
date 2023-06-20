package gpuburn

import (
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nvidiagpu/internal/gpuparams"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	isFalse bool = false
	isTrue  bool = true

	gpuBurnConfigMapData = map[string]string{
		"entrypoint.sh": `#!/bin/bash
		NUM_GPUS=$(nvidia-smi -L | wc -l)
		if [ $NUM_GPUS -eq 0 ]; then
  			echo "ERROR No GPUs found"
			exit 1
		fi
		./gpu_burn 300

		if [ ! $? -eq 0 ]; then
		  exit 1
		fi`,
	}
)

// CreateGPUBurnConfigMap returns a configmap with data field populated.
func CreateGPUBurnConfigMap(apiClient *clients.Settings,
	configMapName, configMapNamespace string) (*corev1.ConfigMap, error) {
	configMapBuilder := configmap.NewBuilder(apiClient, configMapName, configMapNamespace)

	configMapBuilderWithData := configMapBuilder.WithData(gpuBurnConfigMapData)

	createdConfigMapBuilderWithData, err := configMapBuilderWithData.Create()

	if err != nil {
		glog.V(gpuparams.GpuLogLevel).Infof(
			"error creating ConfigMap with Data named %s and for namespace %s",
			createdConfigMapBuilderWithData.Object.Name, createdConfigMapBuilderWithData.Object.Namespace)

		return nil, err
	}

	glog.V(gpuparams.GpuLogLevel).Infof(
		"Created ConfigMap with Data named %s and for namespace %s",
		createdConfigMapBuilderWithData.Object.Name, createdConfigMapBuilderWithData.Object.Namespace)

	return createdConfigMapBuilderWithData.Object, nil
}

// CreateGPUBurnPod returns a Pod after it is Ready after a timeout periods.
func CreateGPUBurnPod(apiClient *clients.Settings, podName, podNamespace string,
	gpuBurnImage string, timeout time.Duration) (*corev1.Pod, error) {
	var volumeDefaultMode int32 = 0777

	configMapVolumeSource := &corev1.ConfigMapVolumeSource{}
	configMapVolumeSource.Name = "gpu-burn-entrypoint"
	configMapVolumeSource.DefaultMode = &volumeDefaultMode

	var err error = nil

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: podNamespace,
			Labels: map[string]string{
				"app": "gpu-burn-app",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			SecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot:   &isTrue,
				SeccompProfile: &corev1.SeccompProfile{Type: "RuntimeDefault"},
			},
			Tolerations: []corev1.Toleration{
				{
					Operator: corev1.TolerationOpExists,
				},
				{
					Key:      "nvidia.com/gpu",
					Effect:   corev1.TaintEffectNoSchedule,
					Operator: corev1.TolerationOpExists,
				},
			},
			Containers: []corev1.Container{
				{
					Image:           gpuBurnImage,
					ImagePullPolicy: corev1.PullAlways,
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: &isFalse,
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{
								"ALL",
							},
						},
					},
					Name: "gpu-burn-ctr",
					Command: []string{
						"/bin/entrypoint.sh",
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							"nvidia.com/gpu": resource.MustParse("1"),
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "entrypoint",
							MountPath: "/bin/entrypoint.sh",
							ReadOnly:  true,
							SubPath:   "entrypoint.sh",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "entrypoint",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: configMapVolumeSource,
					},
				},
			},
			NodeSelector: map[string]string{
				"nvidia.com/gpu.present":         "true",
				"node-role.kubernetes.io/worker": "",
			},
		},
	}, err
}
