package olm

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	pkgManifestV1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/operators/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PackageManifestBuilder provides a struct for PackageManifest object from the cluster
// and a PackageManifest definition.
type PackageManifestBuilder struct {
	// PackageManifest definition. Used to create
	// PackageManifest object with minimum set of required elements.
	Definition *pkgManifestV1.PackageManifest
	// Created PackageManifest object on the cluster.
	Object *pkgManifestV1.PackageManifest
	// api client to interact with the cluster.
	apiClient *clients.Settings
	// errorMsg is processed before PackageManifest object is created.
	errorMsg string
}

// ListPackageManifest returns PackageManifest inventory in the given namespace.
func ListPackageManifest(
	apiClient *clients.Settings,
	nsname string,
	options metaV1.ListOptions) ([]*PackageManifestBuilder, error) {
	glog.V(100).Infof("Listing PackageManifests in the namespace %s with the options %v", nsname, options)

	if nsname == "" {
		glog.V(100).Infof("packagemanifest 'nsname' parameter can not be empty")

		return nil, fmt.Errorf("failed to list packagemanifests, 'nsname' parameter is empty")
	}

	pkgManifestList, err := apiClient.PackageManifestInterface.PackageManifests(nsname).List(context.Background(),
		options)

	if err != nil {
		glog.V(100).Infof("Failed to list PackageManifests in the namespace %s due to %s",
			nsname, err.Error())

		return nil, err
	}

	var pkgManifestObjects []*PackageManifestBuilder

	for _, runningPkgManifest := range pkgManifestList.Items {
		copiedPkgManifest := runningPkgManifest
		pkgManifestBuilder := &PackageManifestBuilder{
			apiClient:  apiClient,
			Object:     &copiedPkgManifest,
			Definition: &copiedPkgManifest,
		}

		pkgManifestObjects = append(pkgManifestObjects, pkgManifestBuilder)
	}

	return pkgManifestObjects, nil
}

// PullPackageManifest loads an existing PackageManifest into Builder struct.
func PullPackageManifest(apiClient *clients.Settings, name, nsname string) (*PackageManifestBuilder, error) {
	glog.V(100).Infof("Pulling existing PackageManifest name %s in namespace %s", name, nsname)

	builder := &PackageManifestBuilder{
		apiClient: apiClient,
		Definition: &pkgManifestV1.PackageManifest{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The Name of the PackageManifest is empty")

		builder.errorMsg = "PackageManifest 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The Namespace of the PackageManifest is empty")

		builder.errorMsg = "PackageManifest 'nsname' cannot be empty"
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("PackageManifest object %s doesn't exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return builder, nil
}

// PullPackageManifestByCatalog loads an existing PackageManifest from specified catalog into Builder struct.
func PullPackageManifestByCatalog(apiClient *clients.Settings, name, nsname,
	catalog string) (*PackageManifestBuilder, error) {
	glog.V(100).Infof("Pulling existing PackageManifest name %s in namespace %s and from catalog %s",
		name, nsname, catalog)

	packageManifests, err := ListPackageManifest(apiClient, nsname, metaV1.ListOptions{
		LabelSelector: fmt.Sprintf("catalog=%s", catalog),
		FieldSelector: fmt.Sprintf("metadata.name=%s", name),
	})

	if err != nil {
		glog.V(100).Infof("Failed to list PackageManifests with name %s in namespace %s from catalog"+
			" %s due to %s", name, nsname, catalog, err.Error())

		return nil, err
	}

	if len(packageManifests) == 0 {
		glog.V(100).Infof("The list of matching PackageManifests is empty")

		return nil, fmt.Errorf("no matching PackageManifests were found")
	}

	if len(packageManifests) > 1 {
		glog.V(100).Infof("More than one matching PackageManifests were found")

		return nil, fmt.Errorf("more than one matching PackageManifests were found")
	}

	return packageManifests[0], nil
}

// Exists checks whether the given PackageManifest exists.
func (builder *PackageManifestBuilder) Exists() bool {
	glog.V(100).Infof(
		"Checking if PackageManifest %s exists", builder.Definition.Name)

	var err error
	builder.Object, err = builder.apiClient.PackageManifestInterface.PackageManifests(
		builder.Definition.Namespace).Get(context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// Delete removes a PackageManifest.
func (builder *PackageManifestBuilder) Delete() error {
	glog.V(100).Infof("Deleting PackageManifest %s in namespace %s", builder.Definition.Name,
		builder.Definition.Namespace)

	if !builder.Exists() {
		return nil
	}

	err := builder.apiClient.PackageManifestInterface.PackageManifests(builder.Definition.Namespace).Delete(
		context.TODO(), builder.Object.Name, metaV1.DeleteOptions{})

	if err != nil {
		return err
	}

	builder.Object = nil

	return err
}
