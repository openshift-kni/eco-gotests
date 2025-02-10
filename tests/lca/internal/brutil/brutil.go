package brutil

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

// NewBackupRestoreObject returns a BackupRestoreObject.
func NewBackupRestoreObject(object runtime.Object, scheme *runtime.Scheme,
	gvr schema.GroupVersion) *BackupRestoreObject {
	return &BackupRestoreObject{
		Scheme: scheme,
		GVR:    gvr,
		Object: object,
	}
}

// String returns a string representation of the object passed to BackupRestoreObject.
func (b *BackupRestoreObject) String() (string, error) {
	if b == nil {
		return "", fmt.Errorf("backuprestoreobject is nil")
	}

	stringRep, err := dataFromDefinition(b.Scheme, b.Object, b.GVR)
	if err != nil {
		return "", err
	}

	return stringRep, nil
}

// dataFromDefinition returns a json string representation of the provided resource.
func dataFromDefinition(scheme *runtime.Scheme, obj runtime.Object, version schema.GroupVersion) (string, error) {
	codec := serializer.NewCodecFactory(scheme).LegacyCodec(version)

	data, err := runtime.Encode(codec, obj)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
