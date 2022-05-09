package agent

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
)

const (
	registerServiceEndpoint   = "/v1/catalog/register"
	deregisterServiceEndpoint = "/v1/catalog/deregister"
)

func doRequest(logger *logrus.Logger, url string, body io.Reader) error {
	req, err := http.NewRequest(http.MethodPut, url, body)
	if err != nil {
		return err
	}

	logger.Debugf("do %s request: %s", req.Method, req.URL.String())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	logger.Debugf("response body: %s", string(b))
	if resp.StatusCode/100 > 2 {
		return fmt.Errorf("Expect 2xx but got %s", http.StatusText(resp.StatusCode))
	}
	return nil
}

// https://www.consul.io/api/agent/service.html#parameters-2
type service struct {
	ID      string
	Name    string `json:"Service"`
	Tags    []string
	Address string
	Meta    map[string]string
	Port    int
}

type registerServiceRequest struct {
	Datacenter string `json:",omitempty"`
	Node       string
	Address    string
	// NodeMeta       map[string]string
	Service        *service
	SkipNodeUpdate bool
}

type deregisterServiceRequest struct {
	Datacenter string `json:",omitempty"`
	Node       string
	ServiceID  string
}
