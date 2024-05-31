package url

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/golang/glog"
)

// Fetch retrieves specified URL using GET or HEAD method.
func Fetch(url, method string) (string, int, error) {
	var (
		res *http.Response
		err error
	)

	switch {
	case strings.EqualFold(method, "get"):
		glog.V(90).Infof("Attempt to retrieve %v with %s method\n", url, strings.ToUpper(method))
		res, err = http.Get(url)
	case strings.EqualFold(method, "head"):
		glog.V(90).Infof("Attempt to retrieve %v with %s method\n", url, strings.ToUpper(method))
		res, err = http.Head(url)
	default:
		glog.Warning(fmt.Sprintf("Unsupported method: %v\n", method))

		return "", 0, fmt.Errorf("unsupported method %v", method)
	}

	if err != nil {
		glog.Warning(fmt.Sprintf("Error accessing %s ; Reason %v\n", url, err))

		return "", 0, fmt.Errorf("error accessing %s ; Reason %w", url, err)
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)

	if err != nil {
		glog.Warning(fmt.Sprintf("Error reading reply: %v\n", err))

		return "", 0, fmt.Errorf("error reading reply: %w", err)
	}

	glog.V(50).Infof("\tReply: %s\n", body)
	glog.V(50).Infof("\tStatus: %s\n", res.Status)

	for key, value := range res.Header {
		glog.V(50).Infof("  %v: %v\n", key, value)
	}

	return string(body), res.StatusCode, nil
}
