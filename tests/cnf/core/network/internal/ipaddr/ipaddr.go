package ipaddr

import "strings"

// RemovePrefix removed prefix form ip address.
func RemovePrefix(ipAddr string) string {
	return strings.Split(ipAddr, "/")[0]
}
