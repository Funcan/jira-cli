// Package jira is a client for accessing JIRA instances.
package jira

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// InstanceConfig describes how to reach and authenticate to a JIRA instance.
type InstanceConfig struct {
	Instance string `yaml:"instance"`
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Token    string `yaml:"token"`
}

// ProjectConfig describes a JIRA project and the instance it lives on.
//
// The field lists name fields by ID or display name, case-insensitively.
type ProjectConfig struct {
	Project  string   `yaml:"project"`
	Instance string   `yaml:"instance"`
	Teams    []string `yaml:"teams"`
	// ShowFields are the fields shown by `jira show`.
	ShowFields []string `yaml:"show_fields"`
	// EditFields are the fields in the `jira edit` template.
	EditFields []string `yaml:"edit_fields"`
	// CreateFields are the fields in the `jira create` template.
	CreateFields []string `yaml:"create_fields"`
	// CreateDefaults pre-fills fields in the `jira create` template.
	CreateDefaults map[string]any `yaml:"create_defaults"`
}

// Defaults holds the project and team to use when none is specified.
type Defaults struct {
	Project string `yaml:"project"`
	Team    string `yaml:"team"`
}

// Config is the contents of ~/.jira.yaml.
type Config struct {
	Projects  []ProjectConfig  `yaml:"projects"`
	Instances []InstanceConfig `yaml:"instances"`
	Defaults  Defaults         `yaml:"defaults"`
}

// DefaultConfigPath returns the path to ~/.jira.yaml.
func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".jira.yaml"), nil
}

// LoadConfig reads and parses the config file at path. If path is empty,
// DefaultConfigPath is used.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		var err error
		if path, err = DefaultConfigPath(); err != nil {
			return nil, err
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	return &cfg, nil
}

// Project returns the config for the named project, or the default project
// if name is empty.
func (c *Config) Project(name string) (*ProjectConfig, error) {
	if name == "" {
		name = c.Defaults.Project
	}
	if name == "" {
		return nil, fmt.Errorf("no project specified and no default project configured")
	}
	for i := range c.Projects {
		if c.Projects[i].Project == name {
			return &c.Projects[i], nil
		}
	}
	return nil, fmt.Errorf("project %q not found in config", name)
}

// Instance returns the config for the named instance.
func (c *Config) Instance(name string) (*InstanceConfig, error) {
	for i := range c.Instances {
		if c.Instances[i].Instance == name {
			return &c.Instances[i], nil
		}
	}
	return nil, fmt.Errorf("instance %q not found in config", name)
}
