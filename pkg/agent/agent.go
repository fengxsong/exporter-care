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
	"sync"
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
	// states
	done chan struct{}
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
		done:          make(chan struct{}, 1),
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

func (a *Agent) watchSignal(sigCh chan os.Signal) {
	var once sync.Once
	for sig := range sigCh {
		a.logger.Info("🤯 Oops! received signal: ", sig)
		if err := a.runPreStopHooks(); err != nil {
			a.logger.Errorf("Error occurred while executing prestophooks: %v", err)
		}
		once.Do(func() {
			if a.cmd.Process != nil && a.cmd.ProcessState == nil {
				if err := a.cmd.Process.Signal(sig); err != nil {
					a.logger.Errorf("Failed to sending signal to process: %v", err)
				}
			}
		})
	}
	a.done <- struct{}{}
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
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go a.watchSignal(sigCh)
	errCh := make(chan error, 1)
	go func() {
		errCh <- a.cmd.Run()
	}()
	defer func() {
		if err != nil {
			sigCh <- syscall.SIGKILL
		}
		close(sigCh)
	}()
	select {
	case err = <-errCh:
		return
	// todo: pass through when process had been started
	case <-time.After(3 * time.Second):
		a.logger.Info("🚀 Looks like everything's fine")
	}
	if err = a.runPostStartHooks(); err != nil {
		return
	}
	select {
	case err = <-errCh:
		return
	}
}

// Wait todo...
func (a *Agent) Wait() error {
	<-a.done
	return nil
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
