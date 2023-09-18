package search

import "github.com/golang/glog"

// StringInSlice check if match string is exist in string slice/array.
func StringInSlice(match string, listOfString []string) bool {
	glog.V(100).Infof("verify that %s exist %v", match, listOfString)

	for _, str := range listOfString {
		if str == match {
			return true
		}
	}

	return false
}
