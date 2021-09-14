package config

import (
	"fmt"
	"gopkg.in/yaml.v2"
)

type ChurroAdminConfig struct {
	DataSource Source `yaml:"dataSource"`
}

func (c ChurroAdminConfig) String() string {
	d, err := yaml.Marshal(&c)
	if err != nil {
		return err.Error()
	}
	return string(d)
}

func GetAdminConfig(configString string) (config ChurroAdminConfig, err error) {

	b := []byte(configString)
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return config, fmt.Errorf("error in unmarshalling admin config %v", err)
	}
	return config, err
}

func GetAdminConfigFromPath(configPath string) (config ChurroAdminConfig, err error) {
	b, err := getConfigBytes(configPath)
	if err != nil {
		return config, err
	}

	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return config, fmt.Errorf("error in unmarshalling admin config %v", err)
	}
	return config, nil
}
