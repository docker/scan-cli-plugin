package main

import (
	"fmt"

	"github.com/docker/docker-scan/config"

	"github.com/docker/cli/cli-plugins/manager"
	"github.com/docker/cli/cli-plugins/plugin"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker-scan/internal"
	"github.com/docker/docker-scan/internal/provider"
	"github.com/spf13/cobra"
)

func main() {
	plugin.Run(func(dockerCli command.Cli) *cobra.Command {
		return newScanCmd(dockerCli)
	}, manager.Metadata{
		SchemaVersion: "0.1.0",
		Vendor:        "Docker Inc.",
		Version:       internal.Version,
	})
}

func newScanCmd(_ command.Cli) *cobra.Command {
	var (
		showVersion bool
	)
	cmd := &cobra.Command{
		Short:       "Docker Scan",
		Long:        `A tool to scan your docker image`,
		Use:         "scan [OPTIONS]",
		Annotations: map[string]string{},
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := config.ReadConfigFile()
			if err != nil {
				return err
			}
			scanProvider := provider.NewSnykProvider(conf.Path)
			if showVersion {
				version, err := internal.FullVersion(scanProvider)
				if err != nil {
					return err
				}
				fmt.Println(version)
				return nil
			}
			if err := cmd.Usage(); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&showVersion, "version", false, "Display version of scan plugin and snyk cli")
	return cmd
}
