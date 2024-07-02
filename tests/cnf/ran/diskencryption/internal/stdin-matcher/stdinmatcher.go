// Original code from https://github.com/greyerof/stdin-matcher

package stdinmatcher

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"time"
)

// Matcher struc to hold regex to match for and the minimum of matches expected.
type Matcher struct {
	// Line regexp.
	Regex *regexp.Regexp
	// Num times (lines) that we're waiting the regex to match.
	Times int
}

// WaitForAnyMatch waits for expected regex matches
// if function found a match, the first match index is return as index in the match slice
// If there was not match after timeout duration, timedout is set to true.
func WaitForAnyMatch(input io.Reader, matches []Matcher, timeout time.Duration) (index int, timedout bool, err error) {
	reader := bufio.NewReader(input)

	errCh := make(chan error)
	lineCh := make(chan string)

	go func() {
		line := ""

		for {
			data, isPrefix, err := reader.ReadLine()
			if err != nil {
				errCh <- err

				break
			}

			line += string(data)
			if !isPrefix {
				lineCh <- line
				line = ""
			}
		}
	}()

	timeoutCh := time.After(timeout)

	matchingCount := make([]int, len(matches))

	for {
		select {
		case <-timeoutCh:
			return 0, true, nil
		case err := <-errCh:
			return 0, false, err
		case line := <-lineCh:
			for inti := range matches {
				regexText := matches[inti]

				match := regexText.Regex.Find([]byte(line))
				if match != nil {
					matchingCount[inti]++
				}

				if matchingCount[inti] == regexText.Times {
					return inti, false, nil
				}
			}
		}
	}
}

// BMC is an interface used to abstract a BMC api that is able to open a console
// This is used by the unit test.
type BMC interface {
	OpenSerialConsole(openConsoleCliCmd string) (io.Reader, io.WriteCloser, error)
	CloseSerialConsole() error
}

// WaitForRegex waits for any matches passed in a matches slice to appear in the
// BMC console for the expected number of times.
// If no matches, timedout is set to true.
func WaitForRegex(bmc BMC, timeout time.Duration, matches []Matcher) (matchIndex int, timedout bool, err error) {
	reader, _, err := bmc.OpenSerialConsole("")
	if err != nil {
		return matchIndex, false, err
	}

	defer func() {
		err = bmc.CloseSerialConsole()
		if err != nil {
			fmt.Printf("error closing BMC serial console, err: %s", err)
		}
	}()

	return WaitForAnyMatch(reader, matches, timeout)
}
