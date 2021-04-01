package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	Folder    string     `yaml:"folder"`
	Mode      string     `yaml:"mode"`
	Filter    string     `yaml:"filter"`
	Port      uint       `yaml:"port"`
	TLSConfig *TLSConfig `yaml:"tls"`
}

type TLSConfig struct {
	Enabled bool   `yaml:"enabled"`
	Key     string `yaml:"key,omitempty"`
	Cert    string `yaml:"cert,omitempty"`
}

func LoadConfig(path string, c *Config) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(b, c)
	if err != nil {
		return err
	}
	return nil
}
