package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/docker/cli/cli-plugins/manager"
	"github.com/docker/cli/cli-plugins/plugin"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker-scan/config"
	"github.com/docker/docker-scan/internal"
	"github.com/docker/docker-scan/internal/authentication"
	"github.com/docker/docker-scan/internal/provider"
	registrytypes "github.com/docker/docker/api/types/registry"
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

type options struct {
	authenticate   bool
	dependencyTree bool
	dockerFilePath string
	excludeBase    bool
	jsonFormat     bool
	showVersion    bool
}

func newScanCmd(dockerCli command.Cli) *cobra.Command {
	var flags options
	cmd := &cobra.Command{
		Short:       "Docker Scan",
		Long:        `A tool to scan your docker image`,
		Use:         "scan [OPTIONS] IMAGE",
		Annotations: map[string]string{},
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.showVersion {
				return runVersion(dockerCli, flags)
			}
			if flags.authenticate {
				return runAuthentication(dockerCli, flags, args)
			}
			return runScan(cmd, dockerCli, flags, args)
		},
	}
	cmd.Flags().BoolVar(&flags.authenticate, "auth", false, "Authenticate to the scan provider using an optional token, or web base token if empty")
	cmd.Flags().BoolVar(&flags.dependencyTree, "dependency-tree", false, "Show dependency tree before scan results")
	cmd.Flags().BoolVar(&flags.excludeBase, "exclude-base", false, "Exclude base image from vulnerabiliy scanning (needs to provide a Dockerfile using --file)")
	cmd.Flags().StringVarP(&flags.dockerFilePath, "file", "f", "", "Provide the Dockerfile for better scan results")
	cmd.Flags().BoolVar(&flags.jsonFormat, "json", false, "Display results with JSON format")
	cmd.Flags().BoolVar(&flags.showVersion, "version", false, "Display version of scan plugin")

	return cmd
}

func configureProvider(dockerCli command.Cli, flags options, options ...provider.SnykProviderOps) (provider.Provider, error) {
	conf, err := config.ReadConfigFile()
	if err != nil {
		return nil, err
	}
	opts := []provider.SnykProviderOps{provider.WithPath(conf.Path)}
	opts = append(opts, options...)
	if flags.jsonFormat {
		opts = append(opts, provider.WithJSON())
	}
	if flags.dockerFilePath != "" {
		opts = append(opts, provider.WithDockerFile(flags.dockerFilePath))
		if flags.excludeBase {
			opts = append(opts, provider.WithoutBaseImageVulnerabilities())
		}
	} else if flags.excludeBase {
		return nil, fmt.Errorf("--file flag is mandatory to use --exclude-base flag")
	}
	if flags.dependencyTree {
		opts = append(opts, provider.WithDependencyTree())
	}
	return provider.NewSnykProvider(opts...)
}

func runVersion(dockerCli command.Cli, flags options) error {
	scanProvider, err := configureProvider(dockerCli, flags)
	if err != nil {
		return err
	}

	version, err := internal.FullVersion(scanProvider)
	if err != nil {
		return err
	}
	fmt.Println(version)
	return nil
}

func runAuthentication(dockerCli command.Cli, flags options, args []string) error {
	scanProvider, err := configureProvider(dockerCli, flags)
	if err != nil {
		return err
	}
	token := ""
	switch {
	case len(args) == 1:
		token = args[0]
	case len(args) > 1:
		return fmt.Errorf(`--auth flag expects maximum one argument`)
	}
	return scanProvider.Authenticate(token)
}

func runScan(cmd *cobra.Command, dockerCli command.Cli, flags options, args []string) error {
	if len(args) != 1 {
		if err := cmd.Usage(); err != nil {
			return err
		}
		return fmt.Errorf(`"docker scan" requires exactly 1 argument`)
	}
	token, err := getToken(dockerCli)
	if err != nil {
		return fmt.Errorf("failed to get DockerScanID: %s", err)
	}

	scanProvider, err := configureProvider(dockerCli, flags, provider.WithDockerScanIDToken(token))
	if err != nil {
		return err
	}
	err = scanProvider.Scan(args[0])
	if exitError, ok := err.(*exec.ExitError); ok {
		os.Exit(exitError.ExitCode())
	}
	return err
}

func getToken(dockerCli command.Cli) (string, error) {
	hubAuthConfig := command.ResolveAuthConfig(context.Background(), dockerCli, hub)
	if hubAuthConfig.Username == "" {
		return "", fmt.Errorf(`You need to be logged in to Docker Hub to use scan feature.
please login to Docker Hub using the Docker Login command`)
	}
	authenticator := authentication.NewAuthenticator(jwksStaging)
	return authenticator.GetToken(hubAuthConfig)
}

var hub = &registrytypes.IndexInfo{
	Name:     "docker.io",
	Mirrors:  nil,
	Secure:   true,
	Official: true,
}

const (
	jwksStaging = `{
  "keys": [
    {
      "use": "sig",
      "kty": "EC",
      "kid": "yy49bsZVoCPg6PgH1iXtuBlOAMVPsMpNb78iUvqrTn/3iDmS6N5nPVjtpcZqgXyAUl4S6tbihdSSPk3nTsGOxA==",
      "crv": "P-256",
      "alg": "ES256",
      "x": "NjptJx3r6yRl895HksB9pK6UmxGZgRMznkRzQCAnHbg",
      "y": "RuuhcGfpxiNZ8__hGRkzc-TGxMVOVWThNEj1-tL_Sk0"
    }
  ]
}`
	jwksProd = `{
  "keys": [
    {
      "use": "sig",
      "kty": "EC",
      "kid": "/Il5tHgzaqqjh6vp1Je9pG0Ic+s/eRQ7C1dLkmITuop0z8qLNszOuqIJldWSEPitEN/cCW5BKt0buUoVHy9o6A==",
      "crv": "P-256",
      "alg": "ES256",
      "x": "oWouB0UC--Gg7hhYiOKExx2dXVsSdP4t7xfIYbVVXSI",
      "y": "b7WeNOKN2Ur00AFO-8-1o_hdflRCz9gtq-JE-3dFvRU"
    }
  ]
}`
)
