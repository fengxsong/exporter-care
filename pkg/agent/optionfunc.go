package agent

import "net"

// OptionFunc ..
type OptionFunc func(a *Agent)

// WithService override service
func WithService(s string) OptionFunc {
	return func(a *Agent) {
		a.service = s
	}
}

// WithOverride override service id
func WithOverride(s string) OptionFunc {
	return func(a *Agent) {
		a.override = s
	}
}

// WithTags ..
func WithTags(tags []string) OptionFunc {
	return func(a *Agent) {
		a.tags = tags
	}
}

// WithMeta ..
func WithMeta(m map[string]string) OptionFunc {
	return func(a *Agent) {
		a.meta = m
	}
}

// WithDatacenter ..
func WithDatacenter(dc string) OptionFunc {
	return func(a *Agent) {
		a.datacenter = dc
	}
}

// WithAdvertiseIP override advertise ip address
func WithAdvertiseIP(ip net.IP) OptionFunc {
	return func(a *Agent) {
		a.advertiseIP = ip.String()
	}
}
