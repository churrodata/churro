// Package config implements the churro configuration file definition
// and helper functions for dealing with the churro configuration
package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type Endpoint struct {
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
	Scheme string `yaml:"scheme"`
}
type WatchRule struct {
	ColumnName string `yaml:"columnname"`
	RuleScript string `yaml:"rulescript"`
}

type WatchSocket struct {
	Name      string   `yaml:"name"`
	Path      string   `yaml:"path"`
	Scheme    string   `yaml:"scheme"`
	Stocks    []string `yaml:"stocks"`
	Tablename string   `yaml:"tablename"`
}
type WatchDirectory struct {
	Name      string      `yaml:"name"`
	Path      string      `yaml:"path"`
	Scheme    string      `yaml:"scheme"`
	Regex     string      `yaml:"regex"`
	Tablename string      `yaml:"tablename"`
	Rules     []WatchRule `yaml:"rules"`
}
type Source struct {
	Name      string `yaml:"name"`
	Host      string `yaml:"host"`
	Path      string `yaml:"path"`
	Port      int    `yaml:"port"`
	Scheme    string `yaml:"scheme"`
	Username  string `yaml:"username"`
	Database  string `yaml:"database"`
	Tablename string `yaml:"tablename"`
}

type TransformRule struct {
	Path     string `yaml:"path"`
	Scheme   string `yaml:"scheme"`
	Function string `yaml:"function"`
}

type TransformFunction struct {
	Name string `yaml:"name"`
	Src  string `yaml:"src"`
}

type ChurroConfig struct {
	// PipelineName is a unique identifier
	PipelineName string `yaml:"pipelineName"`
	// ServiceCredsSecret is the name of the secret that hold grpc creds
	ServiceCredsSecret string `yaml:"serviceCredsSecret"`
	// DbCredsSecret is the name of the secret that holds db client creds
	DbCredsSecret string `yaml:"dbCredsSecret"`
	// DbNodeCredsSecret is the name of the secret that holds db node creds
	DbNodeCredsSecret string        `yaml:"dbNodeCredsSecret"`
	WatchSockets      []WatchSocket `yaml:"watchSockets"`
	// WatchDirectory is a list of directories to watch
	WatchDirectories []WatchDirectory `yaml:"watchDirectories"`
	// DataSource is the churro data store itself
	AdminDataSource Source `yaml:"adminDataSource"`
	DataSource      Source `yaml:"dataSource"`
	WatchConfig     struct {
		Location Endpoint `yaml:"location"`
	} `yaml:"watchConfig"`
	ExtractConfig struct {
		QueueSize   int      `yaml:"queueSize"`
		PctHeadRoom int      `yaml:"pctHeadRoom"`
		DataSource  Source   `yaml:"dataSource"`
		Location    Endpoint `yaml:"location"`
	} `yaml:"extractConfig"`
	TransformConfig struct {
		Location    Endpoint            `yaml:"location"`
		QueueSize   int                 `yaml:"queueSize"`
		PctHeadRoom int                 `yaml:"pctHeadRoom"`
		Rules       []TransformRule     `yaml:"rules"`
		Functions   []TransformFunction `yaml:"functions"`
	} `yaml:"transformConfig"`
}

func (c ChurroConfig) String() string {
	d, err := yaml.Marshal(&c)
	if err != nil {
		fmt.Printf("error in Marshal %s\n", err.Error())
		return ""
	}
	return string(d)
}

func GetConfigFromString(configString string) (config ChurroConfig, err error) {

	b := []byte(configString)
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return config, fmt.Errorf("error in unmarshalling extract config %v", err)
	}
	return config, nil
}

func GetConfigFromPath(configPath string) (config ChurroConfig, err error) {
	b, err := getConfigBytes(configPath)
	if err != nil {
		return config, fmt.Errorf("error getting config from path %v", err)
	}

	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return config, fmt.Errorf("error in unmarshalling extract config %v", err)
	}
	return config, nil
}

func getConfigBytes(config string) (configBytes []byte, err error) {
	if config == "" {
		return configBytes, errors.New("Error: -config flag not set")
	}
	_, err = os.Stat(config)
	if err != nil {
		return configBytes, fmt.Errorf("Error: config path is not a valid file : %w", err)
	}

	configBytes, err = ioutil.ReadFile(config)
	if err != nil {
		return configBytes, fmt.Errorf("Error: error reading config file: %w", err)
	}
	return configBytes, nil
}

func (s Endpoint) URL() string {
	return fmt.Sprintf("%s://%s:%d", s.Scheme, s.Host, s.Port)
}

func (r TransformRule) String() string {
	d, err := yaml.Marshal(&r)
	if err != nil {
		fmt.Printf("error in Marshal %s\n", err.Error())
		return ""
	}
	return string(d)
}

func (f TransformFunction) String() string {
	d, err := yaml.Marshal(&f)
	if err != nil {
		fmt.Printf("error in Marshal %s\n", err.Error())
		return ""
	}
	return string(d)
}
func (s WatchSocket) String() string {
	return fmt.Sprintf("Name      %s \n Path      %s \n Scheme    %s \n Stocks    %v \n Tablename %s", s.Name, s.Path, s.Scheme, s.Stocks, s.Tablename)
}
