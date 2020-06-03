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

func newScanCmd(dockerCli command.Cli) *cobra.Command {
	var (
		showVersion bool
	)
	cmd := &cobra.Command{
		Short:       "Docker Scan",
		Long:        `A tool to scan your docker image`,
		Use:         "scan [OPTIONS] IMAGE",
		Annotations: map[string]string{},
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := config.ReadConfigFile()
			if err != nil {
				return err
			}
			scanProvider := provider.NewSnykProvider(conf.Path, dockerCli.ConfigFile().AuthConfigs)
			if showVersion {
				version, err := internal.FullVersion(scanProvider)
				if err != nil {
					return err
				}
				fmt.Println(version)
				return nil
			}

			if len(args) != 1 {
				if err = cmd.Usage(); err != nil {
					return err
				}
				return fmt.Errorf(`"docker scan" requires exactly 1 argument`)
			}

			if err = scanProvider.Scan(args[0]); err != nil {
				if provider.IsAuthenticationError(err) {
					return fmt.Errorf(`You need to be logged in to Docker Hub to use scan feature.
please login to Docker Hub using the Docker Login command`)
				}
				return err
			}

			return nil
		},
	}
	cmd.Flags().BoolVar(&showVersion, "version", false, "Display version of scan plugin and snyk cli")
	return cmd
}
