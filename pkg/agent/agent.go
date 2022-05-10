package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/multierr"
)

type hook func() error

// Agent ..
type Agent struct {
	// os command
	cmd *exec.Cmd
	// logger
	logger *logrus.Logger
	// agent info
	port        int
	nodeName    string
	advertiseIP string
	service     string
	override    string
	tags        []string
	meta        map[string]string
	// consul servers info
	consulCluster []*url.URL
	datacenter    string
	// hooks
	postStartHooks []hook
	preStopHooks   []hook
}

// New todo..
func New(cmd *exec.Cmd, port int, consulCluster []string, logger *logrus.Logger, funcs ...OptionFunc) (*Agent, error) {
	nodeName, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	localIP, err := getExternalIP()
	if err != nil {
		return nil, err
	}
	var urls []*url.URL
	for i := range consulCluster {
		u, err := url.Parse(consulCluster[i])
		if err != nil {
			return nil, err
		}
		urls = append(urls, u)
	}
	if logger == nil {
		logger = logrus.New()
	}

	a := &Agent{
		cmd:           cmd,
		logger:        logger,
		port:          port,
		nodeName:      nodeName,
		advertiseIP:   localIP,
		consulCluster: urls,
	}
	for _, fn := range funcs {
		fn(a)
	}
	a.setDefault()
	return a, nil
}

func (a *Agent) setDefault() {
	if a.service == "" {
		a.service = filepath.Base(a.cmd.Path)
	}
}

// AddPostStartHook ..
func (a *Agent) AddPostStartHook(h hook) {
	a.postStartHooks = append(a.postStartHooks, h)
}

// AddPreStopHook ..
func (a *Agent) AddPreStopHook(h hook) {
	a.preStopHooks = append(a.preStopHooks, h)
}

func (a *Agent) runPostStartHooks() error {
	var errs []error
	if len(a.postStartHooks) > 0 {
		for i := range a.postStartHooks {
			if err := a.postStartHooks[i](); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return multierr.Combine(errs...)
	}
	return nil
}

func (a *Agent) runPreStopHooks() error {
	var errs []error
	if len(a.preStopHooks) > 0 {
		for i := range a.preStopHooks {
			if err := a.preStopHooks[i](); err != nil {
				errs = append(errs, err)
			}
		}
		// only run pre stop hooks once
		a.preStopHooks = nil
	}
	if len(errs) > 0 {
		return multierr.Combine(errs...)
	}
	return nil
}

func (a *Agent) getConsulEndpoint(path string) string {
	var u *url.URL
	for {
		index := rand.Intn(len(a.consulCluster))
		u = a.consulCluster[index]
		if u.Host != "" {
			break
		}
	}
	u.Path = path
	return u.String()
}

// Register ..
func (a *Agent) Register() error {
	if !contains(a.tags, a.service) {
		a.tags = append(a.tags, a.service)
	}
	req := registerServiceRequest{
		Node:       a.nodeName,
		Address:    a.advertiseIP,
		Datacenter: a.datacenter,
		Service: &service{
			Name:    a.service,
			Tags:    a.tags,
			Address: a.advertiseIP,
			Meta:    a.meta,
			Port:    a.port,
		},
		SkipNodeUpdate: true,
	}
	if a.override != "" {
		req.Service.ID = strings.ReplaceAll(strings.ToLower(a.override), "/", "_")
	} else {
		req.Service.ID = fmt.Sprintf("%s:%d", a.advertiseIP, a.port)
	}
	b, err := json.Marshal(&req)
	if err != nil {
		return err
	}
	return doRequest(a.logger, a.getConsulEndpoint(registerServiceEndpoint), bytes.NewBuffer(b))
}

// DeRegister ..
func (a *Agent) DeRegister() error {
	req := deregisterServiceRequest{
		Node:       a.nodeName,
		Datacenter: a.datacenter,
	}
	if a.override != "" {
		req.ServiceID = strings.ReplaceAll(strings.ToLower(a.override), "/", "_")
	} else {
		req.ServiceID = fmt.Sprintf("%s:%d", a.advertiseIP, a.port)
	}
	b, err := json.Marshal(&req)
	if err != nil {
		return err
	}
	return doRequest(a.logger, a.getConsulEndpoint(deregisterServiceEndpoint), bytes.NewBuffer(b))
}

// Run ..
func (a *Agent) Run() (err error) {
	a.logger.Debugf("command args: %v, envs: %v", a.cmd.Args, a.cmd.Env)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.cmd.Run()
	}()

	select {
	case err = <-errCh:
		return err
	case <-sigCh:
		return nil
	case <-time.After(3 * time.Second):
		a.logger.Info("Seems like everything's fine")
	}
	// command starts successfully
	defer a.cmd.Process.Kill()

	if err = a.runPostStartHooks(); err != nil {
		return err
	}

	defer a.runPreStopHooks()
	select {
	case err = <-errCh:
		return err
	case sig := <-sigCh:
		a.logger.Infof("Receive signal %s, exiting gracefully", sig)
		return nil
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
