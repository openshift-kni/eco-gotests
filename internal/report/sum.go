package main

import (
	"crypto/sha256"
	"embed"
	"fmt"
)

//go:embed *.go
var programSourceCode embed.FS

// sourceCodeSum is the SHA256 sum of this program's source code. Since a change to this program's source code may
// result in a change to the cache format, all cache entries that do not match this sum will be expired. This tends to
// expire cache too eagerly but guarantees compatibility.
var sourceCodeSum string = getSourceSum()

// getSourceSum returns the SHA256 sum of all the .go files in this directory. If any errors are encountered, an empty
// string is returned.
func getSourceSum() string {
	summer := sha256.New()

	dirEntries, err := programSourceCode.ReadDir(".")
	if err != nil {
		return ""
	}

	for _, dirEntry := range dirEntries {
		contents, err := programSourceCode.ReadFile(dirEntry.Name())
		if err != nil {
			return ""
		}

		_, err = summer.Write(contents)
		if err != nil {
			return ""
		}
	}

	return fmt.Sprintf("%x", summer.Sum(nil))
}
