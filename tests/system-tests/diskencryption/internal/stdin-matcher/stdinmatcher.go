package stdinmatcher

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"regexp"
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
func WaitForAnyMatch(ctx context.Context, input io.Reader, matches []Matcher) (int, error) {
	reader := bufio.NewReader(input)
	lineCh := make(chan string)
	errCh := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case lineCh <- scanner.Text():
			}
		}

		if err := scanner.Err(); err != nil {
			errCh <- err
		}

		close(lineCh)
	}()

	matchingCount := make([]int, len(matches))

	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case err := <-errCh:
			return 0, err
		case line, ok := <-lineCh:
			if !ok {
				return 0, io.EOF
			}

			for i, m := range matches {
				if m.Regex.MatchString(line) {
					matchingCount[i]++
					if matchingCount[i] == m.Times {
						return i, nil
					}
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
func WaitForRegex(ctx context.Context, bmc BMC, matches []Matcher) (int, error) {
	reader, _, err := bmc.OpenSerialConsole("")
	if err != nil {
		return 0, err
	}

	defer func() {
		err = bmc.CloseSerialConsole()
		if err != nil {
			fmt.Printf("error closing BMC serial console, err: %s", err)
		}
	}()

	return WaitForAnyMatch(ctx, reader, matches)
}
