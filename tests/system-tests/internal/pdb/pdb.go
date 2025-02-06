package pdb

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/poddisruptionbudget"
	v1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
)

// RetrieveActivePDBMap retrieves active PDB list and returns map of the pdb objects and original minAvailable values.
func RetrieveActivePDBMap(
	apiClient *clients.Settings) (map[*poddisruptionbudget.Builder]*intstr.IntOrString, error) {
	glog.V(100).Infof("Retrieve active PodDisruptionBudget list")

	allAvailablePDB, err := poddisruptionbudget.ListInAllNamespaces(apiClient)

	if err != nil {
		glog.Infof("Failed to list all available pod disruption budget due to %v", err)

		return nil, fmt.Errorf("failed to list all available pod disruption budget due to %w", err)
	}

	pdbMinAvailableCollection := make(map[*poddisruptionbudget.Builder]*intstr.IntOrString)

	for _, _pdb := range allAvailablePDB {
		minAvailableValue := _pdb.Object.Spec.MinAvailable

		if minAvailableValue != nil && minAvailableValue.IntValue() >= 1 {
			pdbMinAvailableCollection[_pdb] = minAvailableValue
		}
	}

	if len(pdbMinAvailableCollection) == 0 {
		glog.V(100).Infof("no poddisruptionbudget with the MinAvailable >= 1 found")

		return nil, fmt.Errorf("no poddisruptionbudget with the MinAvailable >= 1 found")
	}

	return pdbMinAvailableCollection, nil
}

// SetMinAvailableToZeroForActivePDB sets the minAvailable value to zero and returns map from
// the pdb object and original minAvailable value.
func SetMinAvailableToZeroForActivePDB(
	apiClient *clients.Settings) (map[*poddisruptionbudget.Builder]*intstr.IntOrString, error) {
	glog.V(100).Infof("Workaround for the node drain failure due to active PodDisruptionBudget")

	pdbMinAvailableCollection, err := RetrieveActivePDBMap(apiClient)

	if err != nil {
		glog.Infof("Failed to retrieve all active PodDisruptionBudgets due to %v", err)

		return nil, fmt.Errorf("failed to list all active PodDisruptionBudgets due to %w", err)
	}

	zeroInt := intstr.FromInt32(0)

	setZeroValue := v1.PodDisruptionBudgetSpec{
		MinAvailable: &zeroInt,
	}

	for _pdb, value := range pdbMinAvailableCollection {
		err = wait.PollUntilContextTimeout(
			context.TODO(),
			time.Second,
			time.Second*3,
			true,
			func(ctx context.Context) (bool, error) {
				if *value != zeroInt {
					_, err = _pdb.WithPDBSpec(setZeroValue).Update(false)

					if err != nil {
						glog.V(100).Infof("Failed to update PodDisruptionBudget due to %v", err)

						return false, nil
					}
				}

				return true, nil
			})

		if err != nil {
			glog.V(100).Infof("Failed to update PodDisruptionBudget due to %v", err)

			return nil, fmt.Errorf("failed to update PodDisruptionBudget due to %w", err)
		}
	}

	return pdbMinAvailableCollection, nil
}

// RestoreActivePDBValues restores original minAvailable values based on the data from the provided map.
func RestoreActivePDBValues(
	apiClient *clients.Settings,
	pdbMinAvailableCollection map[*poddisruptionbudget.Builder]*intstr.IntOrString) error {
	glog.V(100).Infof(
		"Workaround for the node drain failure due to active PodDisruptionBudget, restore active PDBs values")

	for _pdb, value := range pdbMinAvailableCollection {
		err := wait.PollUntilContextTimeout(
			context.TODO(),
			time.Second,
			time.Second*3,
			true,
			func(ctx context.Context) (bool, error) {
				updatePDB, err := poddisruptionbudget.Pull(apiClient, _pdb.Definition.Name, _pdb.Definition.Namespace)

				if err != nil {
					glog.V(100).Infof("Failed to retrieve PodDisruptionBudget %s in namespace %s due to %v",
						_pdb.Definition.Name, _pdb.Definition.Namespace, err)

					return false, nil
				}

				currentMinAvailableValue := updatePDB.Object.Spec.MinAvailable

				if currentMinAvailableValue != nil && currentMinAvailableValue.IntValue() == 0 {
					restoreValue := v1.PodDisruptionBudgetSpec{
						MinAvailable: value,
					}
					_, err := updatePDB.WithPDBSpec(restoreValue).Update(false)

					if err != nil {
						glog.V(100).Infof("Failed to update PodDisruptionBudget due to %v", err)

						return false, nil
					}
				}

				return true, nil
			})

		if err != nil {
			glog.V(100).Infof("Failed to retrieve updated PodDisruptionBudget %s in namespace %s due to %v",
				_pdb.Definition.Name, _pdb.Definition.Namespace, err)

			return fmt.Errorf("failed to retrieve updated PodDisruptionBudget %s in namespace %s due to %w",
				_pdb.Definition.Name, _pdb.Definition.Namespace, err)
		}
	}

	return nil
}
