package config

import (
	"errors"
	"os"
)

type ServiceCredentials struct {
	ServiceCrt string
	ServiceKey string
}

func (s ServiceCredentials) Validate() error {
	_, err := os.Stat(s.ServiceCrt)
	if err != nil {
		return errors.New("-servicecrt flag required")
	}

	_, err = os.Stat(s.ServiceKey)
	if err != nil {
		return errors.New("-servicekey flag required")
	}

	return nil

}
