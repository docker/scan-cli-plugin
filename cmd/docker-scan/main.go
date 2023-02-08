/*
   Copyright 2020 Docker Inc.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/docker/cli/cli-plugins/manager"
	"github.com/docker/cli/cli-plugins/plugin"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/scan-cli-plugin/config"
	"github.com/docker/scan-cli-plugin/internal"
	"github.com/docker/scan-cli-plugin/internal/optin"
	"github.com/docker/scan-cli-plugin/internal/provider"
	"github.com/spf13/cobra"
)

func main() {
	ctx, closeFunc := newSigContext()
	defer closeFunc()
	plugin.Run(func(dockerCli command.Cli) *cobra.Command {
		cmd := newScanCmd(ctx, dockerCli)
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
	login          bool
	token          string
	dependencyTree bool
	dockerFilePath string
	excludeBase    bool
	jsonFormat     bool
	showVersion    bool
	forceOptIn     bool
	forceOptOut    bool
	severity       string
	groupIssues    bool
}

func newScanCmd(ctx context.Context, dockerCli command.Cli) *cobra.Command {
	var flags options
	cmd := &cobra.Command{
		Short:       "Docker Scan",
		Long:        `A tool to scan your images`,
		Use:         "scan [OPTIONS] IMAGE",
		Annotations: map[string]string{},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !flags.jsonFormat {
				internal.PrintDeprecationMessage(dockerCli)
			}

			if flags.showVersion {
				return runVersion(ctx, dockerCli, flags)
			}
			if flags.login {
				return runAuthentication(ctx, dockerCli, flags, args)
			}
			return runScan(ctx, cmd, dockerCli, flags, args)
		},
	}
	cmd.Flags().BoolVar(&flags.login, "login", false, "Authenticate to the scan provider using an optional token (with --token), or web base token if empty")
	cmd.Flags().StringVar(&flags.token, "token", "", "Authentication token to login to the third party scanning provider")
	cmd.Flags().BoolVar(&flags.dependencyTree, "dependency-tree", false, "Show dependency tree with scan results")
	cmd.Flags().BoolVar(&flags.excludeBase, "exclude-base", false, "Exclude base image from vulnerability scanning (requires --file)")
	cmd.Flags().StringVarP(&flags.dockerFilePath, "file", "f", "", "Dockerfile associated with image, provides more detailed results")
	cmd.Flags().BoolVar(&flags.jsonFormat, "json", false, "Output results in JSON format")
	cmd.Flags().BoolVar(&flags.showVersion, "version", false, "Display version of the scan plugin")
	cmd.Flags().BoolVar(&flags.forceOptIn, "accept-license", false, "Accept using a third party scanning provider")
	cmd.Flags().BoolVar(&flags.forceOptOut, "reject-license", false, "Reject using a third party scanning provider")
	cmd.Flags().StringVar(&flags.severity, "severity", "", "Only report vulnerabilities of provided level or higher (low|medium|high)")
	cmd.Flags().BoolVar(&flags.groupIssues, "group-issues", false, "Aggregate duplicated vulnerabilities and group them to a single one (requires --json)")

	return cmd
}

func configureProvider(ctx context.Context, dockerCli command.Cli, flags options, options ...provider.Ops) (provider.Provider, error) {
	conf, err := checkConsent(flags, dockerCli)
	if err != nil {
		return nil, err
	}

	opts := []provider.Ops{
		provider.WithContext(ctx),
		provider.WithPath(conf.Path),
	}
	opts = append(opts, options...)
	if flags.jsonFormat {
		opts = append(opts, provider.WithJSON())
		opts = append(opts, provider.WithExperimental())
		if flags.groupIssues {
			opts = append(opts, provider.WithGroupIssues())
		}
	} else if flags.groupIssues {
		return nil, fmt.Errorf("--json flag is mandatory to use --group-issues flag")
	}
	opts = append(opts, provider.WithAppVulns())

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
	if flags.severity != "" {
		if flags.severity != "low" && flags.severity != "medium" && flags.severity != "high" {
			return nil, fmt.Errorf("--severity takes only 'low', 'medium' or 'high' values")
		}
		opts = append(opts, provider.WithSeverity(flags.severity))
	}
	defaultProvider, err := provider.NewProvider(opts...)
	if err != nil {
		return nil, err
	}
	if runtime.GOOS == "linux" && !provider.UseExternalBinary(defaultProvider) {
		return provider.NewDockerSnykProvider(dockerCli, defaultProvider)
	}
	return provider.NewSnykProvider(defaultProvider)
}

func checkConsent(flags options, dockerCli command.Streams) (config.Config, error) {
	conf, err := config.ReadConfigFile()
	if err != nil {
		return config.Config{}, err
	}
	if flags.showVersion {
		return conf, nil
	}

	if !conf.Optin || flags.forceOptIn || flags.forceOptOut {
		switch {
		case !flags.forceOptOut && !flags.forceOptIn:
			conf.Optin = optin.AskForConsent(dockerCli.In(), dockerCli.Out())
		case flags.forceOptOut && flags.forceOptIn:
			return config.Config{}, fmt.Errorf("enable and disable flags are mutualy exlusive")
		case flags.forceOptIn:
			conf.Optin = true
		case flags.forceOptOut:
			conf.Optin = false
		}

		if err := config.SaveConfigFile(conf); err != nil {
			return config.Config{}, err
		}
		if !conf.Optin {
			os.Exit(0)
		}
	}
	return conf, nil
}

func runVersion(ctx context.Context, dockerCli command.Cli, flags options) error {
	scanProvider, err := configureProvider(ctx, dockerCli, flags)
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

func runAuthentication(ctx context.Context, dockerCli command.Cli, flags options, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf(`--login flag expects no argument`)
	}
	scanProvider, err := configureProvider(ctx, dockerCli, flags)
	if err != nil {
		return err
	}
	return scanProvider.Authenticate(flags.token)
}

func runScan(ctx context.Context, cmd *cobra.Command, dockerCli command.Cli, flags options, args []string) error {
	scanProvider, err := configureProvider(ctx, dockerCli, flags, provider.WithAuthConfig(func(hub *registry.IndexInfo) types.AuthConfig {
		return command.ResolveAuthConfig(context.Background(), dockerCli, hub)
	}), provider.WithVersion(internal.Version))
	if len(args) != 1 {
		if err := cmd.Usage(); err != nil {
			return err
		}
		return fmt.Errorf(`"docker scan" requires exactly 1 argument`)
	}
	if err != nil {
		return err
	}
	err = scanProvider.Scan(args[0])

	if !flags.jsonFormat {
		internal.PrintDeprecationMessage(dockerCli)
	}

	if _, ok := err.(*exec.ExitError); ok {
		os.Exit(1)
	}
	return err
}

func newSigContext() (context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	s := make(chan os.Signal)
	signal.Notify(s, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-s
		cancel()
	}()
	return ctx, cancel
}
