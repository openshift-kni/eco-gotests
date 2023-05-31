package polarion

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
)

const (
	polarionTag = "polarion-testcase-id"
	testIDTag   = "test_id"
)

type (
	// TestSuite represents polarion formatted test suite.
	TestSuite struct {
		XMLName    xml.Name   `xml:"testsuite"`
		Name       string     `xml:"name,attr"`
		Tests      int        `xml:"tests,attr"`
		Skipped    int        `xml:"skipped,attr"`
		Failures   int        `xml:"failures,attr"`
		Time       float64    `xml:"time,attr"`
		Properties Properties `xml:"properties"`
		TestCases  []TestCase `xml:"testcase"`
	}

	// TestCase represents polarion formatted test cast.
	TestCase struct {
		Name           string          `xml:"name,attr"`
		Properties     Properties      `xml:"properties"`
		FailureMessage *FailureMessage `xml:"failure,omitempty"`
		Skipped        *Skipped        `xml:"skipped,omitempty"`
		SystemOut      string          `xml:"system-out,omitempty"`
	}

	// FailureMessage represents polarion fail message.
	FailureMessage struct {
		Type    string `xml:"type,attr"`
		Message string `xml:",chardata"`
	}

	// Skipped represents polarion skip message.
	Skipped struct {
		XMLName xml.Name `xml:"skipped"`
		Message string   `xml:"message,attr,omitempty"`
	}

	// Properties structure represents polarion test case properties.
	Properties struct {
		Property []Property `xml:"property"`
	}

	// Property represents polarion test case property.
	Property struct {
		Name  string `xml:"name,attr"`
		Value string `xml:"value,attr"`
	}
)

// CreateReport writes polarion report to a given xml file.
func CreateReport(report ginkgo.Report, destFile, projectTag string) {
	if destFile == "" {
		return
	}

	testSuite := setTestSuite(report)

	for _, testCaseSpecReport := range report.SpecReports {
		if testCaseSpecReport.FullText() == "" {
			continue
		}

		testCase := TestCase{
			Name: testCaseSpecReport.FullText(),
		}

		if polarionID := setPolarionID(testCaseSpecReport, projectTag); polarionID != nil {
			testCase.Properties.Property = append(testCase.Properties.Property, *polarionID)
		}

		if failedMessage := setFailureMessage(testCaseSpecReport); failedMessage != nil {
			testCase.FailureMessage = failedMessage
		}

		if skippedMessage := setSkipMessage(testCaseSpecReport); skippedMessage != nil {
			testCase.Skipped = skippedMessage
		}

		testSuite.TestCases = append(testSuite.TestCases, testCase)
		testSuite.Tests++
	}

	generatePolarionXMLFile(destFile, testSuite)
}

// ID sets polarion id for a test case.
func ID(tag string) ginkgo.Labels {
	return ginkgo.Label(tag, fmt.Sprintf("%s:%s", testIDTag, tag))
}

// SetProperty sets polarion property for a test case.
func SetProperty(propertyKey, propertyValue string) ginkgo.Labels {
	return ginkgo.Label(fmt.Sprintf("polarion-parameter-%s:%s", propertyKey, propertyValue))
}

func setPolarionID(testReport types.SpecReport, projectTag string) *Property {
	if len(testReport.Labels()) > 0 {
		for _, label := range testReport.Labels() {
			if strings.Contains(label, testIDTag) {
				return &Property{
					Name:  polarionTag,
					Value: fmt.Sprintf("%s%s", projectTag, strings.Split(label, ":")[1]),
				}
			}
		}
	}

	return nil
}

func setFailureMessage(testReport types.SpecReport) *FailureMessage {
	if types.SpecStateFailureStates.Is(testReport.State) {
		return &FailureMessage{
			Type:    failureTypeForState(testReport.State),
			Message: failureMessage(testReport.Failure),
		}
	}

	return nil
}

func setSkipMessage(testReport types.SpecReport) *Skipped {
	if types.SpecStateSkipped.Is(testReport.State) {
		return &Skipped{
			XMLName: xml.Name{Space: testReport.Failure.Message},
			Message: testReport.Failure.Message,
		}
	}

	return nil
}

func setTestSuite(report ginkgo.Report) *TestSuite {
	return &TestSuite{
		XMLName:  xml.Name{Space: report.SuiteDescription},
		Name:     report.SuiteDescription,
		Tests:    0,
		Time:     report.RunTime.Seconds(),
		Skipped:  report.SpecReports.CountWithState(types.SpecStateSkipped),
		Failures: report.SpecReports.CountWithState(types.SpecStateFailureStates),
	}
}

func generatePolarionXMLFile(outputFile string, testCases interface{}) {
	file, err := os.Create(outputFile)
	if err != nil {
		panic(fmt.Errorf("failed to create Polarion report file: %s\n\t%w", outputFile, err))
	}

	defer func() {
		_ = file.Close()
	}()

	_, _ = file.WriteString(xml.Header)
	encoder := xml.NewEncoder(file)
	encoder.Indent("  ", "    ")
	err = encoder.Encode(testCases)

	if err != nil {
		panic(fmt.Errorf("failed to generate Polarion report\n\t%w", err))
	}
}

func failureTypeForState(state types.SpecState) string {
	switch state {
	case types.SpecStateFailed:
		return "Failure"
	case types.SpecStateInterrupted:
		return "Interrupted"
	case types.SpecStatePanicked:
		return "Panic"
	default:
		return ""
	}
}

func failureMessage(failure types.Failure) string {
	return fmt.Sprintf(
		"%s\n%s\n%s", failure.FailureNodeLocation.String(), failure.Message, failure.Location.String())
}
