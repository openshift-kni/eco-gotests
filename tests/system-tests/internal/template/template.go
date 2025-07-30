package template

import (
	"fmt"
	"os"
	"text/template"

	"github.com/golang/glog"
)

// SaveTemplate read template file, replace variables and save it to the provided destination folder.
func SaveTemplate(
	source,
	destination string,
	variablesToReplace map[string]interface{}) error {
	if source == "" {
		glog.V(100).Infof("The source is empty")

		return fmt.Errorf("the source should be provided")
	}

	glog.V(100).Infof("Read %s template, replace variables and save it locally to the %s",
		source, destination)

	if destination == "" {
		glog.V(100).Infof("The destination file is empty")

		return fmt.Errorf("the destination file should be provided")
	}

	tmpl, err := template.ParseFiles(source)
	if err != nil {
		glog.V(100).Infof("Error to read config file %s", source)

		return err
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
		glog.V(100).Infof("Error to chmod file %s", destination)

		return err
	}

	return nil
}
