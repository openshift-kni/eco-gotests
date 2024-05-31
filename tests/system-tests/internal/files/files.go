package files

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
)

// DownloadFile downloads file from the provided url locally;
// if fileName is provided, it will be saved under provided name.
func DownloadFile(fileURL, fileName, targetFolder string) error {
	if fileURL == "" {
		glog.V(100).Info("The fileURL is empty")

		return fmt.Errorf("the fileURL should be provided")
	}

	if fileName == "" {
		glog.V(100).Info("Build fileName from the fullPath")

		fileURL, err := url.Parse(fileURL)

		if err != nil {
			return fmt.Errorf("failed to parse URL %s due to: %w", fileURL, err)
		}

		path := fileURL.Path
		segments := strings.Split(path, "/")
		fileName = segments[len(segments)-1]
	}

	// Get the data
	resp, err := http.Get(fileURL)
	if err != nil {
		return fmt.Errorf("failed to get data from the URL %s due to: %w", fileURL, err)
	}
	defer resp.Body.Close()

	// Create the file
	filePath := filepath.Join(targetFolder, fileName)
	out, err := os.Create(filePath)

	if err != nil {
		return fmt.Errorf("failed to create file %s at folder %s due to: %w", fileName, targetFolder, err)
	}

	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)

	return err
}
