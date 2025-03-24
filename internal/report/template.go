package main

import (
	_ "embed"
	"html/template"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

var (
	//go:embed tree_template.html
	treeTemplateFile string

	//go:embed report_template.html
	reportTemplateFile string
)

var (
	funcMap = template.FuncMap{
		"cleanPath": cleanPath,
	}
	treeTemplate   = template.Must(template.New("tree_template.html").Funcs(funcMap).Parse(treeTemplateFile))
	reportTemplate = template.Must(template.New("report_template.html").Parse(reportTemplateFile))
)

// TreeTemplateConfig contains the data necessary to template a single SuiteTree into an html report.
type TreeTemplateConfig struct {
	Tree       *SuiteTree
	Generated  time.Time
	Branch     string
	ActionURL  template.URL
	RepoURL    template.URL
	TimeFormat string
}

// TemplateTree uses config to generate a SuiteTree report and save it at outputFileName.
func TemplateTree(config TreeTemplateConfig, outputFileName string) error {
	return executeTemplateAndSave(treeTemplate, config, outputFileName)
}

// ReportTemplateConfig contains the data necessary to generate a report linking to multiple templated SuiteTrees.
type ReportTemplateConfig struct {
	BranchReports []BranchReportConfig
	Generated     time.Time
	ActionURL     template.URL
	RepoURL       template.URL
	TimeFormat    string
}

// BranchReportConfig contains the data necessary to include a single templated SuiteTree for a certain branch.
type BranchReportConfig struct {
	Name          string
	ReportFile    string
	Revision      string
	ShortRevision string
}

// TemplateReport uses config to generate a report linking to multiple SuiteTree reports and save it at outputFileName.
// Branch reports will be sorted lexicographically ascending by branch name.
func TemplateReport(config ReportTemplateConfig, outputFileName string) error {
	slices.SortFunc(config.BranchReports, func(a, b BranchReportConfig) int {
		return strings.Compare(a.Name, b.Name)
	})

	return executeTemplateAndSave(reportTemplate, config, outputFileName)
}

// executeTemplateAndSave creates a file at outputFileName before executing tmpl with data provided by config. If
// outputFileName already exists, then it is truncated.
func executeTemplateAndSave(tmpl *template.Template, config any, outputFileName string) error {
	outputFile, err := os.Create(outputFileName)
	if err != nil {
		return err
	}

	defer outputFile.Close()

	err = tmpl.Execute(outputFile, config)
	if err != nil {
		return err
	}

	return nil
}

// cleanPath cleans the provided path of anything preceding the eco-gotests directory. This is useful to template paths
// to be relative to the repo root rather than / on the machine that generated the report.
func cleanPath(path string) string {
	pathElements := strings.Split(path, string(os.PathSeparator))
	for i, element := range pathElements {
		if element == "eco-gotests" {
			return filepath.Join(pathElements[i:]...)
		}
	}

	return path
}
