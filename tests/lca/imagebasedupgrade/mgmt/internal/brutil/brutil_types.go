package brutil

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// BackupRestoreObject contains information for generating
// string representations of API resources.
type BackupRestoreObject struct {
	Scheme *runtime.Scheme
	GVR    schema.GroupVersion
	Object runtime.Object
}
