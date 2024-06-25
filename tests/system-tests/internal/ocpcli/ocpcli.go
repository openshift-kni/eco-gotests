package ocpcli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/files"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/platform"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/shell"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/template"
	"github.com/walle/targz"
)

// DownloadAndExtractOcBinaryArchive downloads and extracts oc binary archive.
func DownloadAndExtractOcBinaryArchive(apiClient *clients.Settings) error {
	ocBinaryMirror := "https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp"
	localTarArchiveName := "openshift-client-linux.tar.gz"
	tempDir := os.TempDir()

	ocPath := filepath.Join(tempDir, "oc")

	glog.V(100).Info("Check if oc binary already downloaded locally")

	if stat, err := os.Stat(ocPath); err == nil && stat.Name() != "" {
		glog.V(100).Info("oc binary was found, need to be deleted")

		err = os.Remove(ocPath)

		if err != nil {
			return fmt.Errorf("failed to remove %s", ocPath)
		}
	}

	glog.V(100).Info("install oc binary")

	clusterVersion, err := platform.GetOCPVersion(apiClient)

	if err != nil {
		return err
	}

	ocBinaryURL := fmt.Sprintf("%s/%s/openshift-client-linux-%s.tar.gz",
		ocBinaryMirror, clusterVersion, clusterVersion)

	err = files.DownloadFile(ocBinaryURL, localTarArchiveName, tempDir)

	if err != nil {
		return err
	}

	tarArchiveLocation := filepath.Join(tempDir, localTarArchiveName)
	err = targz.Extract(tarArchiveLocation, tempDir)

	if err != nil {
		return err
	}

	execCmd := fmt.Sprintf("sudo -u root -i cp %s /usr/local/bin/oc", ocPath)
	output, err := shell.ExecuteCmd(execCmd)

	if err != nil {
		return fmt.Errorf("failed to execute %s command due to: %w. \noutput: %v", execCmd, err, output)
	}

	chmodCmd := "sudo -u root -i chmod 755 /usr/local/bin/oc"
	_, err = shell.ExecuteCmd(chmodCmd)

	if err != nil {
		return fmt.Errorf("failed to execute %s command due to: %w", chmodCmd, err)
	}

	return nil
}

// ApplyConfig applies config using shell method.
func ApplyConfig(
	templateDir,
	fileName,
	destinationDir,
	finalFileName string,
	variablesToReplace map[string]interface{}) error {
	err := template.SaveTemplate(
		templateDir, fileName, destinationDir, finalFileName, variablesToReplace)

	if err != nil {
		return err
	}

	cfgFilePath := filepath.Join(destinationDir, finalFileName)

	applyCmd := fmt.Sprintf("oc apply -f %s", cfgFilePath)
	_, err = shell.ExecuteCmd(applyCmd)

	if err != nil {
		return fmt.Errorf("failed to execute %s command due to: %w", applyCmd, err)
	}

	return nil
}

// CreateConfig creates config using shell method.
func CreateConfig(
	templateDir,
	fileName,
	destinationDir,
	finalFileName string,
	variablesToReplace map[string]interface{}) error {
	err := template.SaveTemplate(
		templateDir, fileName, destinationDir, finalFileName, variablesToReplace)

	if err != nil {
		return err
	}

	cfgFilePath := filepath.Join(destinationDir, finalFileName)

	applyCmd := fmt.Sprintf("oc create -f %s", cfgFilePath)
	_, err = shell.ExecuteCmd(applyCmd)

	if err != nil {
		return fmt.Errorf("failed to execute %s command due to: %w", applyCmd, err)
	}

	return nil
}

// PatchWithNamespace patches the resource using the given patch type, object belongs to the specific namespace
// The following patches are exactly the same patch but using different types, 'merge' and 'json'
// --type merge -p '{"spec": {"selector": {"app": "frommergepatch"}}}'
// --type json  -p '[{ "op": "replace", "path": "/spec/selector/app", "value": "fromjsonpatch"}]'.
func PatchWithNamespace(objName, objNamespace, objKind, patchType, patchStr string) error {
	patchCmd := fmt.Sprintf("oc patch %s/%s -n %s --type %s -p '%v'",
		objKind, objName, objNamespace, patchType, patchStr)

	_, err := shell.ExecuteCmd(patchCmd)
	if err != nil {
		return fmt.Errorf("failed to execute %s command due to: %w", patchCmd, err)
	}

	return nil
}

// PatchWithoutNamespace patches the resource using the given patch type, object has no namespace
// The following patches are exactly the same patch but using different types, 'merge' and 'json'
// --type merge -p '{"spec": {"selector": {"app": "frommergepatch"}}}'
// --type json  -p '[{ "op": "replace", "path": "/spec/selector/app", "value": "fromjsonpatch"}]'.
func PatchWithoutNamespace(objName, objKind, patchType, patchStr string) error {
	patchCmd := fmt.Sprintf("oc patch %s/%s --type %s -p '%v'",
		objKind, objName, patchType, patchStr)

	_, err := shell.ExecuteCmd(patchCmd)
	if err != nil {
		return fmt.Errorf("failed to execute %s command due to: %w", patchCmd, err)
	}

	return nil
}

// AddClusterRoleToServiceAccount adds specific cluster role to the serviceaccount.
func AddClusterRoleToServiceAccount(serviceAccountName, namespace, clusterRole string) error {
	patchCmd := fmt.Sprintf("oc adm policy add-cluster-role-to-user %s -z %s -n '%s'",
		clusterRole, serviceAccountName, namespace)

	_, err := shell.ExecuteCmd(patchCmd)
	if err != nil {
		return fmt.Errorf("failed to execute %s command due to: %w", patchCmd, err)
	}

	return nil
}

// ExecuteViaDebugPodOnNode creates debug pod and executes provided command.
func ExecuteViaDebugPodOnNode(
	nodeName string,
	cmd string) (string, error) {
	execCmd := fmt.Sprintf("oc debug node/%s -- bash -c \"chroot /host %s\" --insecure-skip-tls-verify",
		nodeName, cmd)
	glog.V(100).Infof("Execute command %s", execCmd)

	output, err := shell.ExecuteCmd(execCmd)
	if err != nil {
		return "", fmt.Errorf("failed to execute %s command due to: %w", execCmd, err)
	}

	return string(output), nil
}
