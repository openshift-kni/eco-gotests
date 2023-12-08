package template

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/golang/glog"
)

// SaveTemplate read template file, replace variables and save it to the provided destination folder.
func SaveTemplate(
	templateDir,
	fileName,
	destinationDir,
	finalFileName string,
	variablesToReplace map[string]interface{}) error {
	if fileName == "" {
		glog.V(100).Infof("The filename is empty")

		return fmt.Errorf("the filename should be provided")
	}

	if finalFileName == "" {
		finalFileName = fileName
	}

	glog.V(100).Infof("Read %s template, replace variables and save it locally to the %s/%s",
		fileName, destinationDir, finalFileName)

	if templateDir == "" {
		glog.V(100).Infof("The template folder is empty")

		return fmt.Errorf("the template folder should be provided")
	}

	pathToTemplate := filepath.Join(templateDir, fileName)
	destination := filepath.Join(destinationDir, finalFileName)
	tmpl, err := template.ParseFiles(pathToTemplate)

	if err != nil {
		glog.V(100).Infof("Error to read config file %s", pathToTemplate)

		return err
	}

	glog.V(100).Infof("create %s folder if not exists", destinationDir)

	err = os.Mkdir(destinationDir, 0755)

	if err != nil && !os.IsExist(err) {
		return err
	}

	err = os.Remove(destination)

	if err != nil && !os.IsExist(err) {
		glog.V(100).Infof("%s file not found", destination)
	}

	// create a new file
	file, err := os.Create(destination)
	if err != nil {
		glog.V(100).Infof("Error to create file %s", destination)

		return err
	}

	glog.V(100).Infof("apply the template %s to the vars map and write the result to file", destination)

	err = tmpl.Execute(file, variablesToReplace)

	if err != nil {
		glog.V(100).Infof("Error to apply the template to the vars map and write the result to file %s",
			destination)

		return err
	}

	err = os.Chmod(destination, 0755)

	if err != nil {
		glog.V(100).Infof("Error to chmod file %", destination)

		return err
	}

	return nil
}
