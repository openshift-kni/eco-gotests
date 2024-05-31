package kmm

import (
	"fmt"

	"github.com/golang/glog"
	moduleV1Beta1 "github.com/rh-ecosystem-edge/kernel-module-management/api/v1beta1"
	v1 "k8s.io/api/core/v1"
)

// KernelMappingBuilder builds kernelMapping struct based on given parameters.
type KernelMappingBuilder struct {
	// Module definition. Used to create a Module object.
	definition *moduleV1Beta1.KernelMapping
	// Used in functions that define or mutate Module definition. errorMsg is processed before the Module
	// object is created.
	errorMsg string
}

// NewRegExKernelMappingBuilder creates new kernel mapping element based on regex.
func NewRegExKernelMappingBuilder(regex string) *KernelMappingBuilder {
	glog.V(100).Infof(
		"Initializing new regex KernelMapping parameter structure with the following regex param: %s", regex)

	builder := KernelMappingBuilder{
		definition: &moduleV1Beta1.KernelMapping{
			Regexp: regex,
		},
	}

	if regex == "" {
		glog.V(100).Infof("The regex of NewRegExKernelMappingBuilder is empty")

		builder.errorMsg = "'regex' parameter can not be empty"
	}

	return &builder
}

// NewLiteralKernelMappingBuilder create new kernel mapping element based on literal.
func NewLiteralKernelMappingBuilder(literal string) *KernelMappingBuilder {
	glog.V(100).Infof(
		"Initializing new literal KernelMapping parameter structure with following literal param: %s", literal)

	builder := KernelMappingBuilder{
		definition: &moduleV1Beta1.KernelMapping{
			Literal: literal,
		},
	}

	if literal == "" {
		glog.V(100).Infof("The literal of NewLiteralKernelMappingBuilder is empty")

		builder.errorMsg = "'literal' parameter can not be empty"
	}

	return &builder
}

// BuildKernelMappingConfig returns kernel mapping config if error is not occur.
func (builder *KernelMappingBuilder) BuildKernelMappingConfig() (*moduleV1Beta1.KernelMapping, error) {
	glog.V(100).Infof(
		"Returning the KernelMappingBuilder structure %v", builder.definition)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf("error building KernelMappingConfig config due to :%s", builder.errorMsg)
	}

	return builder.definition, nil
}

// WithContainerImage adds the specified Container Image config to the KernelMapper.
func (builder *KernelMappingBuilder) WithContainerImage(image string) *KernelMappingBuilder {
	glog.V(100).Infof(
		"Creating new Module KernelMapping parameter with container image: %s", image)

	if image == "" {
		glog.V(100).Infof("The image of WithContainerImage is empty")

		builder.errorMsg = "'image' parameter can not be empty for KernelMapping"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.definition.ContainerImage = image

	return builder
}

// WithBuildArg adds the specified Build Args config to the KernelMapper.
func (builder *KernelMappingBuilder) WithBuildArg(argName, argValue string) *KernelMappingBuilder {
	glog.V(100).Infof(
		"Creating new Module KernelMapping parameter with buildingArgs name: %s, value: %s", argName, argValue)

	if argName == "" {
		glog.V(100).Infof("The argName of WithBuildArg is empty")

		builder.errorMsg = "'argName' parameter can not be empty for KernelMapping BuildArg"
	}

	if argValue == "" {
		glog.V(100).Infof("The argValue of WithBuildArg is empty")

		builder.errorMsg = "'argValue' parameter can not be empty for KernelMapping BuildArg"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.addBuild()

	builder.definition.Build.BuildArgs = append(
		builder.definition.Build.BuildArgs, moduleV1Beta1.BuildArg{Name: argName, Value: argValue})

	return builder
}

// WithBuildSecret adds the specified Build Secret config to the KernelMapper.
func (builder *KernelMappingBuilder) WithBuildSecret(secret string) *KernelMappingBuilder {
	glog.V(100).Infof(
		"Creating new Module KernelMapping parameter with BuildSecret %s", secret)

	if secret == "" {
		glog.V(100).Infof("The secret of WithBuildSecret is empty")

		builder.errorMsg = "'secret' parameter can not be empty for KernelMapping Secret"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.addBuild()

	builder.definition.Build.Secrets = append(
		builder.definition.Build.Secrets, v1.LocalObjectReference{Name: secret})

	return builder
}

// WithBuildImageRegistryTLS adds the specified ImageRegistryTLS config to the KernelMapper Build.
func (builder *KernelMappingBuilder) WithBuildImageRegistryTLS(insecure, skipTLSVerify bool) *KernelMappingBuilder {
	glog.V(100).Infof(
		"Creating new Module KernelMapping parameter with BuildImageRegistryTLS %t, value: %t",
		insecure, skipTLSVerify)

	if builder.errorMsg != "" {
		return builder
	}

	builder.addBuild()
	builder.definition.Build.BaseImageRegistryTLS.Insecure = insecure
	builder.definition.Build.BaseImageRegistryTLS.InsecureSkipTLSVerify = skipTLSVerify

	return builder
}

// WithBuildDockerCfgFile adds the specified DockerCfgFil config to the KernelMapper Build.
func (builder *KernelMappingBuilder) WithBuildDockerCfgFile(name string) *KernelMappingBuilder {
	glog.V(100).Infof("Creating new Module KernelMapping parameter with DockerCfgFile %s, ", name)

	if name == "" {
		glog.V(100).Infof("The name of WithBuildDockerCfgFile is empty")

		builder.errorMsg = "'name' parameter can not be empty for KernelMapping Docker file"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.addBuild()
	builder.definition.Build.DockerfileConfigMap = &v1.LocalObjectReference{Name: name}

	return builder
}

// WithSign adds the specified Sign config to the KernelMapper.
func (builder *KernelMappingBuilder) WithSign(certSecret, keySecret string, fileToSign []string) *KernelMappingBuilder {
	glog.V(100).Infof(
		"Creating new Module KernelMapping parameter with Sign. CertSecret: %s, KeySecret: %s, fileToSign: %v",
		certSecret, keySecret, fileToSign)

	if certSecret == "" {
		glog.V(100).Infof("The certSecret of WithSign is empty")

		builder.errorMsg = "'certSecret' parameter can not be empty for KernelMapping Sign"
	}

	if keySecret == "" {
		glog.V(100).Infof("The keySecret of WithSign is empty")

		builder.errorMsg = "'keySecret' parameter can not be empty for KernelMapping Sign"
	}

	if len(fileToSign) < 1 {
		glog.V(100).Infof("The fileToSign of WithSign is empty")

		builder.errorMsg = "'fileToSign' parameter can not be empty for KernelMapping Sign"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.definition.Sign = &moduleV1Beta1.Sign{
		CertSecret:  &v1.LocalObjectReference{Name: certSecret},
		KeySecret:   &v1.LocalObjectReference{Name: keySecret},
		FilesToSign: fileToSign,
	}

	return builder
}

// RegistryTLS adds the specified RegistryTLS to the KernelMapper.
func (builder *KernelMappingBuilder) RegistryTLS(insecure, skipTLSVerify bool) *KernelMappingBuilder {
	glog.V(100).Infof(
		"Creating new Module KernelMapping parameter with RegistryTLS. Insecure: %t, InsecureSkipTLSVerify: %t",
		insecure, skipTLSVerify)

	if builder.errorMsg != "" {
		return builder
	}

	builder.definition.RegistryTLS.Insecure = insecure
	builder.definition.RegistryTLS.InsecureSkipTLSVerify = skipTLSVerify

	return builder
}

func (builder *KernelMappingBuilder) addBuild() {
	if builder.definition.Build == nil {
		builder.definition.Build = &moduleV1Beta1.Build{}
	}
}
