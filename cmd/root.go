package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/fengxsong/exporter-care/pkg/agent"
	"github.com/fengxsong/exporter-care/pkg/exporter"
)

var (
	buildDate = "unknown"
	gitCommit = "unknown"
	version   = "unknown"
)

// NewCommand create root command
func NewCommand(name string) *cobra.Command {
	var o = &options{}

	var (
		logLevel string
		logger   *logrus.Logger
	)

	cmd := &cobra.Command{
		Use:   name,
		Short: "Babycare for exporters",
		Long:  "Babycare for exporters which automatic register/deregister service from consul",
		Args:  cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			lvl, err := logrus.ParseLevel(logLevel)
			if err != nil {
				return err
			}
			logger = logrus.New()
			logger.SetLevel(lvl)
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(logger, args[0], o)
		},
		SilenceUsage: true,
	}
	fs := cmd.PersistentFlags()
	fs.StringSliceVar(&o.args, "args", []string{}, "Additional CLI arguments")
	fs.StringSliceVar(&o.env, "env", []string{}, "Additional environment variables")
	fs.StringSliceVar(&o.tags, "tags", []string{}, "A list of tags to assign to the service")
	fs.StringVar(&o.service, "service", "", "Service name registered in consul")
	fs.StringVar(&o.override, "override", "", "Override service id")
	fs.IPVar(&o.advertiseIP, "advertise-ip", nil, "Advertise IPAddr, default is external ip address")
	fs.IntVar(&o.port, "listen-port", 0, "Port to listen on for web interface and telemetry")
	fs.StringSliceVar(&o.consulCluster, "consul-cluster", []string{}, "Address of consul agent(cluster)")
	fs.StringVar(&o.datacenter, "datacenter", "", "Datacenter of the consul agent")
	fs.StringToStringVar(&o.meta, "meta", map[string]string{}, "Arbitrary KV metadata linked to the service instance")

	fs.StringVar(&logLevel, "log-level", "info", "Change log level")

	cmd.MarkFlagRequired("consul-cluster")
	cmd.AddCommand(newVersionCommand())
	setFlagsFromEnv(fs, name)

	return cmd
}

func run(logger *logrus.Logger, name string, o *options) error {
	cmd, port, err := exporter.New(name, o.port, o.args, o.env).Build()
	if err != nil {
		return err
	}
	ag, err := agent.New(cmd, port, o.consulCluster, logger, o.OptionFuncs()...)
	if err != nil {
		return err
	}
	if port != 0 {
		ag.AddPostStartHook(ag.Register)
		ag.AddPreStopHook(ag.DeRegister)
	}
	if err = ag.Run(); err != nil {
		logger.Info("Wait for process to terminated")
		return ag.Wait()
		// if exiterr, ok := err.(*exec.ExitError); ok {
		// 	if _, ok := exiterr.Sys().(syscall.WaitStatus); ok {
		// 		return err
		// 	}
		// }
	}
	logger.Info("Graceful shutdown")
	return nil
}

func setFlagsFromEnv(fs *pflag.FlagSet, prefix string) {
	if prefix != "" {
		prefix += "_"
	}
	fs.VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			return
		}
		envVar := strings.ToUpper(strings.Replace(prefix+f.Name, "-", "_", -1))
		if value := os.Getenv(envVar); value != "" {
			fs.Set(f.Name, value)
		}
	})
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use: "version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("BuildDate: %s, GitCommit: %s, Version: %s, GoVersion: %s, Compiler: %s, Platform: %s",
				buildDate, gitCommit, version, runtime.Version(), runtime.Compiler, fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
		},
	}
}
