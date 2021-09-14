package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/churrodata/churro/api/v1alpha1"
)

type DBCredentials struct {
	CACertPath      string
	CAKeyPath       string
	SSLRootCertPath string
	SSLRootKeyPath  string
	SSLKeyPath      string
	SSLCertPath     string
	Username        string
	Password        string
}

func (d DBCredentials) Validate() error {
	_, err := os.Stat(d.CACertPath)

	if err != nil {
		return errors.New("--cacert flag required")
	}
	_, err = os.Stat(d.CAKeyPath)

	if err != nil {
		return errors.New("--cakey flag required")
	}

	_, err = os.Stat(d.SSLKeyPath)
	if err != nil {
		return errors.New("--dbsslkey flag required")
	}

	_, err = os.Stat(d.SSLCertPath)
	if err != nil {
		return errors.New("--dbsslcert flag required")
	}
	_, err = os.Stat(d.SSLRootKeyPath)
	if err != nil {
		return errors.New("--dbsslrootkey flag required")
	}

	_, err = os.Stat(d.SSLRootCertPath)
	if err != nil {
		return errors.New("--dbsslrootcert flag required")
	}
	return nil

}

func (d DBCredentials) GetDBConnectString(src v1alpha1.Source) string {
	userid := src.Username
	hostname := src.Host
	port := src.Port
	database := src.Database
	sslrootcert := d.CACertPath
	sslkey := d.SSLKeyPath
	sslcert := d.SSLCertPath
	if userid == "root" {
		sslkey = d.SSLRootKeyPath
		sslcert = d.SSLRootCertPath
	}

	str := fmt.Sprintf("postgresql://%s@%s:%d/%s?ssl=true&sslmode=require&sslrootcert=%s&sslkey=%s&sslcert=%s", userid, hostname, port, database, sslrootcert, sslkey, sslcert)
	return str

}
