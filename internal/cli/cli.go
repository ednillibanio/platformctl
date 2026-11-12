package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/ednillibanio/platformctl/internal/config"
	"github.com/ednillibanio/platformctl/internal/platformhome"
	"github.com/ednillibanio/platformctl/internal/render"
	"github.com/ednillibanio/platformctl/internal/validation"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	return NewWithIO(os.Stdout, os.Stderr)
}

func NewWithIO(stdout, stderr io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:           "platformctl",
		Short:         "Manage platform environment configuration",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.AddCommand(initCommand(stdout), environmentCommand(stdout), validateCommand(stdout), renderCommand(stdout), exportCommand(stdout),
		bootstrapConfigCommand(stdout, stderr), credentialCommand(stdout))
	return root
}

func initCommand(stdout io.Writer) *cobra.Command {
	var force bool
	command := &cobra.Command{
		Use:   "init",
		Short: "Initialize the platform configuration home",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			home, err := platformhome.Resolve()
			if err != nil {
				return err
			}
			if err := config.Init(home, force); err != nil {
				return err
			}
			fmt.Fprintf(stdout, "Initialized platform home at %s\n", home)
			return nil
		},
	}
	command.Flags().BoolVar(&force, "force", false, "overwrite divergent managed scaffold files")
	return command
}

func environmentCommand(stdout io.Writer) *cobra.Command {
	parent := &cobra.Command{Use: "environment", Short: "Manage environments"}
	var force bool
	initEnv := &cobra.Command{
		Use:   "init <env>",
		Short: "Initialize an environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			home, err := platformhome.Resolve()
			if err != nil {
				return err
			}
			if err := config.InitEnvironment(home, args[0], force); err != nil {
				return err
			}
			fmt.Fprintf(stdout, "Initialized environment %q at %s\n", args[0], home)
			return nil
		},
	}
	initEnv.Flags().BoolVar(&force, "force", false, "overwrite a divergent managed environment file")
	parent.AddCommand(initEnv)
	return parent
}

func validateCommand(stdout io.Writer) *cobra.Command {
	var env string
	command := &cobra.Command{
		Use:   "validate",
		Short: "Validate an environment",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			bundle, err := load(env)
			if err != nil {
				return err
			}
			if err := validation.Bundle(bundle); err != nil {
				return err
			}
			fmt.Fprintf(stdout, "Environment %q is valid\n", env)
			return nil
		},
	}
	command.Flags().StringVar(&env, "env", "", "environment name (required)")
	_ = command.MarkFlagRequired("env")
	return command
}

func renderCommand(stdout io.Writer) *cobra.Command {
	var env, format string
	var revealSecrets bool
	command := &cobra.Command{
		Use:   "render",
		Short: "Render an effective environment",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			bundle, err := load(env)
			if err != nil {
				return err
			}
			resolved, err := render.Resolve(bundle)
			if err != nil {
				return err
			}
			if !revealSecrets {
				resolved = render.Redact(resolved)
			}
			output, err := render.Encode(resolved, format)
			if err != nil {
				return err
			}
			_, err = stdout.Write(output)
			return err
		},
	}
	command.Flags().StringVar(&env, "env", "", "environment name (required)")
	command.Flags().StringVar(&format, "format", "yaml", "output format: yaml or json")
	command.Flags().BoolVar(&revealSecrets, "reveal-secrets", false, "include inline credential values instead of redacting them")
	_ = command.MarkFlagRequired("env")
	return command
}

func exportCommand(stdout io.Writer) *cobra.Command {
	var env, format, prefix, host string
	var services []string
	var revealSecrets bool
	command := &cobra.Command{
		Use:   "export",
		Short: "Export resolved values as shell environment variables",
		Long: `Export resolved values as shell environment variables.

Without --host, the layers are defaults, the environment, and each --service
in the order given. With --host, they are that host's resolution order.

The output is shell code to evaluate, for example:

  eval "$(platformctl export --env lab --service packer --prefix PM_)"

Credential references are resolved by the evaluating shell, never by
platformctl. If any referenced variable is unset or file is unreadable,
nothing is exported and the evaluation returns a non-zero status.`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if format != "env" {
				return fmt.Errorf("unsupported format %q; expected env", format)
			}
			if host != "" && len(services) > 0 {
				return fmt.Errorf("--host and --service are mutually exclusive")
			}
			bundle, err := load(env)
			if err != nil {
				return err
			}
			var resolved config.RenderedConfig
			if host != "" {
				rendered, err := render.Resolve(bundle)
				if err != nil {
					return err
				}
				if resolved, err = render.HostConfig(rendered, host); err != nil {
					return err
				}
			} else if resolved, err = render.ResolveServices(bundle, services); err != nil {
				return err
			}
			output, err := render.EncodeEnv(resolved, render.EnvOptions{Prefix: prefix, RevealSecrets: revealSecrets})
			if err != nil {
				return err
			}
			_, err = stdout.Write(output)
			return err
		},
	}
	command.Flags().StringVar(&env, "env", "", "environment name (required)")
	command.Flags().StringArrayVar(&services, "service", nil, "service layer to apply, in order (repeatable)")
	command.Flags().StringVar(&host, "host", "", "export one host's resolved configuration instead of services")
	command.Flags().StringVar(&format, "format", "env", "output format: env")
	command.Flags().StringVar(&prefix, "prefix", "", "prefix for every variable name, for example PM_")
	command.Flags().BoolVar(&revealSecrets, "reveal-secrets", false, "export inline credential values instead of refusing them")
	_ = command.MarkFlagRequired("env")
	return command
}

func load(env string) (config.Bundle, error) {
	home, err := platformhome.Resolve()
	if err != nil {
		return config.Bundle{}, err
	}
	return config.Load(home, env)
}
