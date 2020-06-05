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
		cmd := newScanCmd(dockerCli)
		originalPreRun := cmd.PersistentPreRunE
		cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
			if err := plugin.PersistentPreRunE(cmd, args); err != nil {
				return err
			}
			if originalPreRun != nil {
				return originalPreRun(cmd, args)
			}
			return nil
		}
		return cmd
	}, manager.Metadata{
		SchemaVersion: "0.1.0",
		Vendor:        "Docker Inc.",
		Version:       internal.Version,
	})
}

func newScanCmd(dockerCli command.Cli) *cobra.Command {
	var (
		authenticate bool
		showVersion  bool
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
			// --version is set, let's show the version
			if showVersion {
				version, err := internal.FullVersion(scanProvider)
				if err != nil {
					return err
				}
				fmt.Println(version)
				return nil
			}
			// --auth flag is set, we run the authentication
			if authenticate {
				token := ""
				switch {
				case len(args) == 1:
					token = args[0]
				case len(args) > 1:
					return fmt.Errorf(`--auth flag expects maximum one argument`)
				}
				return scanProvider.Authenticate(token)
			}
			// let's run the scan
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
	cmd.Flags().BoolVar(&authenticate, "auth", false, "Authenticate to the scan provider using an optional token, or web base token if empty")
	cmd.Flags().BoolVar(&showVersion, "version", false, "Display version of scan plugin")
	return cmd
}
