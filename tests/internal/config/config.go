package config

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v2"
)

const (
	// PathToConst path to config file with constants.
	PathToConst = "const.yaml"
)

// General type keeps general configuration.
type General struct {
	VerboseLevel string `yaml:"verbose_level" envconfig:"VERBOSE_LEVEL"`
}

// NewConfig returns instance of General config type.
func NewConfig() *General {
	var conf General

	_, filename, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filename)
	confFile := filepath.Join(baseDir, PathToConst)
	err := readFile(&conf, confFile)

	if err != nil {
		return nil
	}

	err = readEnv(&conf)

	if err != nil {
		return nil
	}

	return &conf
}

func readFile(conf *General, cfgfile string) error {
	openedCfgFile, err := os.Open(cfgfile)
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
