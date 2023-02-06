package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/golang/glog"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v2"
)

const (
	// PathToConst path to config file with constants.
	PathToConst = "const.yaml"
)

// General type keeps general configuration.
type General struct {
	ReportsDirAbsPath string `yaml:"reports_dump_dir" envconfig:"REPORTS_DUMP_DIR"`
	VerboseLevel      string `yaml:"verbose_level" envconfig:"VERBOSE_LEVEL"`
	DumpFailedTests   string `yaml:"dump_failed_tests" envconfig:"DUMP_FAILED_TESTS"`
}

// NewConfig returns instance of General config type.
func NewConfig() *General {
	glog.V(90).Infof("Creating new global config struct")
	var conf General

	_, filename, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filename)
	confFile := filepath.Join(baseDir, PathToConst)
	err := readFile(&conf, confFile)

	if err != nil {
		glog.V(90).Infof("Error to read config file %s", confFile)
		return nil
	}

	err = readEnv(&conf)

	if err != nil {
		glog.V(90).Infof("Error to read environment variables")
		return nil
	}

	err = deployReportDir(conf.ReportsDirAbsPath)

	if err != nil {
		glog.V(90).Infof(
			"Error to deploy report directory %s due to %s", conf.ReportsDirAbsPath, err.Error())
		return nil
	}

	return &conf
}

// GetReportPath returns full path to the report file.
func (c *General) GetReportPath(file string) string {
	reportFileName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(filepath.Base(file)))

	return fmt.Sprintf("%s.xml", filepath.Join(c.ReportsDirAbsPath, reportFileName))
}

// GetDumpFailedTestReportLocation returns destination file for failed tests logs.
func (c *General) GetDumpFailedTestReportLocation(file string) string {
	if c.DumpFailedTests == "true" {
		if _, err := os.Stat(c.ReportsDirAbsPath); os.IsNotExist(err) {
			err := os.MkdirAll(c.ReportsDirAbsPath, 0744)
			if err != nil {
				glog.Fatalf("panic: Failed to create report dir due to %s", err)
			}
		}

		dumpFileName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(filepath.Base(file)))

		return filepath.Join(c.ReportsDirAbsPath, fmt.Sprintf("failed_%s", dumpFileName))
	}

	return ""
}

func readFile(conf *General, cfgFile string) error {
	openedCfgFile, err := os.Open(cfgFile)
	if err != nil {
		return err
	}

	defer func() {
		_ = openedCfgFile.Close()
	}()

	decoder := yaml.NewDecoder(openedCfgFile)
	err = decoder.Decode(&conf)

	if err != nil {
		return err
	}

	return nil
}

func readEnv(c *General) error {
	err := envconfig.Process("", c)
	if err != nil {
		return err
	}

	return nil
}

func deployReportDir(dirName string) error {
	_, err := os.Stat(dirName)

	if os.IsNotExist(err) {
		return os.MkdirAll(dirName, 0777)
	}

	return err
}
