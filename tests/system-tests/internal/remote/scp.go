package remote

import (
	"fmt"
	"os"

	"github.com/golang/glog"
	ssh "github.com/povsister/scp"
)

// ScpFileFrom transfers specific file using scp method from remote host.
func ScpFileFrom(source, destination, remoteHostname, remoteHostUsername, remoteHostPass string) error {
	if source == "" {
		glog.V(100).Info("The source is empty")

		return fmt.Errorf("the source could not be empty")
	}

	if destination == "" {
		glog.V(100).Info("The destination is empty")

		return fmt.Errorf("the destination could not be empty")
	}

	if remoteHostname == "" {
		glog.V(100).Info("The remoteHostname is empty")

		return fmt.Errorf("the remoteHostname could not be empty")
	}

	if remoteHostUsername == "" {
		glog.V(100).Info("The remoteHostUsername is empty")

		return fmt.Errorf("the remoteHostUsername could not be empty")
	}

	if remoteHostPass == "" {
		glog.V(100).Info("The remoteHostPass is empty")

		return fmt.Errorf("the remoteHostPass could not be empty")
	}

	glog.V(100).Infof("Verify file %s already exists", destination)

	if _, err := os.Stat(destination); os.IsExist(err) {
		glog.V(100).Infof("File file %s already exists", destination)
	}

	glog.V(100).Infof("Copy file %s locally", source)
	glog.V(100).Info("Build a SSH config from username/password")

	sshConf := ssh.NewSSHConfigFromPassword(remoteHostUsername, remoteHostPass)

	glog.V(100).Infof("Dial SSH to %s", remoteHostname)

	scpClient, err := ssh.NewClient(remoteHostname, sshConf, &ssh.ClientOption{})

	if err != nil {
		return err
	}

	glog.V(100).Infof("Transfer file %s to %s", source, destination)

	err = scpClient.CopyFileFromRemote(source, destination, &ssh.FileTransferOption{})

	if err != nil {
		return err
	}

	glog.V(100).Infof("Insure file %s was transferred", destination)

	if _, err := os.Stat(destination); os.IsNotExist(err) {
		return err
	}

	return nil
}

// ScpFileTo transfers specific file using scp method to remote host.
func ScpFileTo(source, destination, remoteHostname, remoteHostUsername, remoteHostPass string) error {
	if source == "" {
		glog.V(100).Info("The source is empty")

		return fmt.Errorf("the source could not be empty")
	}

	if destination == "" {
		glog.V(100).Info("The destination is empty")

		return fmt.Errorf("the destination could not be empty")
	}

	if remoteHostname == "" {
		glog.V(100).Info("The remoteHostname is empty")

		return fmt.Errorf("the remoteHostname could not be empty")
	}

	if remoteHostUsername == "" {
		glog.V(100).Info("The remoteHostUsername is empty")

		return fmt.Errorf("the remoteHostUsername could not be empty")
	}

	if remoteHostPass == "" {
		glog.V(100).Info("The remoteHostPass is empty")

		return fmt.Errorf("the remoteHostPass could not be empty")
	}

	glog.V(100).Infof("Verify file %s exists", source)

	if _, err := os.Stat(source); !os.IsExist(err) {
		glog.V(100).Infof("File file %s not found", source)
	}

	glog.V(100).Infof("Copy file %s to the %s@%s", source, remoteHostname, destination)
	glog.V(100).Info("Build a SSH config from username/password")

	sshConf := ssh.NewSSHConfigFromPassword(remoteHostUsername, remoteHostPass)

	glog.V(100).Infof("Dial SSH to %s", remoteHostname)

	scpClient, err := ssh.NewClient(remoteHostname, sshConf, &ssh.ClientOption{})

	if err != nil {
		return err
	}

	glog.V(100).Infof("Transfer file %s to %s", source, destination)

	err = scpClient.CopyFileToRemote(source, destination, &ssh.FileTransferOption{})

	return err
}

// ScpDirectoryFrom transfers specific directory using scp method from destination.
func ScpDirectoryFrom(source, destination, remoteHostname, remoteHostUsername, remoteHostPass string) error {
	if source == "" {
		glog.V(100).Info("The source is empty")

		return fmt.Errorf("the source could not be empty")
	}

	if destination == "" {
		glog.V(100).Info("The destination is empty")

		return fmt.Errorf("the destination could not be empty")
	}

	if remoteHostname == "" {
		glog.V(100).Info("The remoteHostname is empty")

		return fmt.Errorf("the remoteHostname could not be empty")
	}

	if remoteHostUsername == "" {
		glog.V(100).Info("The remoteHostUsername is empty")

		return fmt.Errorf("the remoteHostUsername could not be empty")
	}

	glog.V(100).Infof("Copy directory %s locally", source)
	glog.V(100).Infof("Create local directory %s if not exists", destination)

	err := os.Mkdir(destination, 0755)

	if err != nil {
		glog.V(100).Infof("Failed to create directory %s, it is already exists", destination)
	}

	glog.V(100).Info("Build a SSH config from username/password")

	sshConf := ssh.NewSSHConfigFromPassword(remoteHostUsername, remoteHostPass)

	glog.V(100).Infof("Dial SSH to %s", remoteHostname)

	scpClient, err := ssh.NewClient(remoteHostname, sshConf, &ssh.ClientOption{})

	if err != nil {
		return err
	}

	glog.V(100).Infof("recursively copy from remote directory %s to the %s", source, destination)

	err = scpClient.CopyDirFromRemote(source, destination, &ssh.DirTransferOption{})

	if err != nil {
		return err
	}

	glog.V(100).Infof("Insure directory %s was transferred", destination)

	if _, err := os.Stat(destination); os.IsNotExist(err) {
		return err
	}

	return nil
}

// ScpDirectoryTo transfers specific directory using scp method to destination.
func ScpDirectoryTo(source, destination, remoteHostname, remoteHostUsername, remoteHostPass string) error {
	if source == "" {
		glog.V(100).Info("The source is empty")

		return fmt.Errorf("the source could not be empty")
	}

	if destination == "" {
		glog.V(100).Info("The destination is empty")

		return fmt.Errorf("the destination could not be empty")
	}

	if remoteHostname == "" {
		glog.V(100).Info("The remoteHostname is empty")

		return fmt.Errorf("the remoteHostname could not be empty")
	}

	if remoteHostUsername == "" {
		glog.V(100).Info("The remoteHostUsername is empty")

		return fmt.Errorf("the remoteHostUsername could not be empty")
	}

	glog.V(100).Infof("Copy directory %s from %s", source, destination)

	glog.V(100).Info("Build a SSH config from username/password")

	sshConf := ssh.NewSSHConfigFromPassword(remoteHostUsername, remoteHostPass)

	glog.V(100).Infof("Dial SSH to %s", remoteHostname)

	scpClient, err := ssh.NewClient(remoteHostname, sshConf, &ssh.ClientOption{})

	if err != nil {
		return err
	}

	glog.V(100).Infof("recursively copy from directory %s to the %s", source, destination)

	err = scpClient.CopyDirToRemote(source, destination, &ssh.DirTransferOption{})

	if err != nil {
		return err
	}

	glog.V(100).Infof("Insure directory %s was transferred", destination)

	if _, err := os.Stat(destination); os.IsNotExist(err) {
		return err
	}

	return nil
}
