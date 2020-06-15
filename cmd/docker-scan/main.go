package main

import (
	"fmt"
	"os"
	"os/exec"

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
		authenticate   bool
		showVersion    bool
		jsonFormat     bool
		excludeBase    bool
		dockerFilePath string
		dependencyTree bool
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
			opts := []provider.SnykProviderOps{
				provider.WithPath(conf.Path),
				provider.WithAuthConfig(dockerCli.ConfigFile().AuthConfigs)}
			if jsonFormat {
				opts = append(opts, provider.WithJSON())
			}
			if dockerFilePath != "" {
				opts = append(opts, provider.WithDockerFile(dockerFilePath))
				if excludeBase {
					opts = append(opts, provider.WithoutBaseImageVulnerabilities())
				}
			} else if excludeBase {
				return fmt.Errorf("--file flag is mandatory to use --exclude-base flag")
			}
			if dependencyTree {
				opts = append(opts, provider.WithDependencyTree())
			}
			scanProvider, err := provider.NewSnykProvider(opts...)
			if err != nil {
				return err
			}
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

			err = scanProvider.Scan(args[0])
			if provider.IsAuthenticationError(err) {
				return fmt.Errorf(`You need to be logged in to Docker Hub to use scan feature.
please login to Docker Hub using the Docker Login command`)
			}
			if exitError, ok := err.(*exec.ExitError); ok {
				os.Exit(exitError.ExitCode())
			}
			return err
		},
	}
	cmd.Flags().BoolVar(&authenticate, "auth", false, "Authenticate to the scan provider using an optional token, or web base token if empty")
	cmd.Flags().BoolVar(&showVersion, "version", false, "Display version of scan plugin")
	cmd.Flags().BoolVar(&jsonFormat, "json", false, "Display results with JSON format")
	cmd.Flags().StringVarP(&dockerFilePath, "file", "f", "", "Provide the Dockerfile for better scan results")
	cmd.Flags().BoolVar(&excludeBase, "exclude-base", false, "Exclude base image from vulnerabiliy scanning (needs to provide a Dockerfile using --file)")
	cmd.Flags().BoolVar(&dependencyTree, "dependency-tree", false, "Show dependency tree before scan results")

	return cmd
}
