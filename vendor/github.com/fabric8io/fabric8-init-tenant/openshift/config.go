package openshift

import (
	"fmt"
	"net/http"
)

type Config struct {
	MasterURL     string
	MasterUser    string
	Token         string
	HttpTransport *http.Transport
}

func (c Config) WithToken(token string) Config {
	return Config{MasterURL: c.MasterURL, MasterUser: c.MasterUser, Token: token, HttpTransport: c.HttpTransport}
}

type multiError struct {
	Message string
	Errors  []error
}

func (m multiError) Error() string {
	s := m.Message + "\n"
	for _, err := range m.Errors {
		s += fmt.Sprintf("%v\n", err)
	}
	return s
}

func (m *multiError) String() string {
	return m.Error()
}
