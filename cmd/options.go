package cmd

import (
	"net"

	"github.com/fengxsong/exporter-care/pkg/agent"
)

type options struct {
	args, env, tags []string
	meta            map[string]string
	port            int
	service         string
	override        string
	advertiseIP     net.IP
	// consul
	datacenter    string
	consulCluster []string
}

func (o *options) OptionFuncs() []agent.OptionFunc {
	funcs := make([]agent.OptionFunc, 0)
	if o.datacenter != "" {
		funcs = append(funcs, agent.WithDatacenter(o.datacenter))
	}
	if o.service != "" {
		funcs = append(funcs, agent.WithService(o.service))
	}
	if len(o.meta) > 0 {
		funcs = append(funcs, agent.WithMeta(o.meta))
	}
	if o.override != "" {
		funcs = append(funcs, agent.WithOverride(o.override))
	}
	if len(o.tags) > 0 {
		funcs = append(funcs, agent.WithTags(o.tags))
	}
	if o.advertiseIP != nil {
		funcs = append(funcs, agent.WithAdvertiseIP(o.advertiseIP))
	}
	return funcs
}
