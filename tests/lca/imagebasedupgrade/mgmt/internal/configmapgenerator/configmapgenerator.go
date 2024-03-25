package configmapgenerator

import (
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// DataFromDefinition returns a json string representation of the provided resource.
func DataFromDefinition(apiClient *clients.Settings, obj goclient.Object, version schema.GroupVersion) (string, error) {
	codec := serializer.NewCodecFactory(apiClient.Scheme()).LegacyCodec(version)
	data, err := runtime.Encode(codec, obj)

	if err != nil {
		return "", err
	}

	return string(data), nil
}
