package cli

import (
	"io"
	"os"

	"github.com/ednillibanio/platformctl/internal/sharedroot"
	"github.com/spf13/cobra"
)

// defaultDeclaration is used when --declaration is not given and the file
// exists in the current directory.
const defaultDeclaration = "shared-root.yaml"

// usageArgs rejects positional arguments as a usage error (exit 2).
func usageArgs(_ *cobra.Command, args []string) error {
	if len(args) > 0 {
		return sharedroot.Usagef("unexpected argument %q", args[0])
	}
	return nil
}

// usageFlagError turns a flag parsing error into a usage error (exit 2).
func usageFlagError(_ *cobra.Command, err error) error {
	return sharedroot.Usagef("%v", err)
}

func bootstrapConfigCommand(stdout, stderr io.Writer) *cobra.Command {
	var opts sharedroot.BootstrapOptions
	var configDir, importKey string
	command := &cobra.Command{
		Use:   "bootstrap-config",
		Short: "Create the provision-* shared root skeleton of an environment",
		Long: `Create the provision-* shared root skeleton of an environment.

Creates <root>/<env>/ with targets/<t>/vars.sh, targets/<t>/hosts/<c>.sh
(commented placeholders of the declared keys), credentials/target-<t>/<identity>/
and credentials/<c>/. The root is --config-dir, else ASSEMBLEIARR_CONFIG_DIR,
else ~/.config/assembleiarr. It never writes a value and never changes an
existing file: for an existing file that lacks declared keys it prints the
lines to add. Declarations default to ./shared-root.yaml when it exists.

--automation-key generates the ed25519 automation key pair on the first
controller of an environment; --automation-key-import - reads that private
key from standard input on every other controller.

Exit status: 0 success, 1 problem, 2 usage error.`,
		Args: usageArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if opts.Env == "" {
				return sharedroot.Usagef("--env is required")
			}
			root, err := sharedroot.ResolveRoot(configDir)
			if err != nil {
				return err
			}
			opts.Root = root
			if cmd.Flags().Changed("automation-key-import") {
				if importKey != "-" {
					return sharedroot.Usagef("--automation-key-import takes only - (the private key is read from standard input)")
				}
				opts.ImportKey = cmd.InOrStdin()
			}
			if len(opts.Declarations) == 0 {
				if info, err := os.Stat(defaultDeclaration); err == nil && info.Mode().IsRegular() {
					opts.Declarations = []string{defaultDeclaration}
				}
			}
			opts.Out, opts.Err = stdout, stderr
			return sharedroot.Bootstrap(opts)
		},
	}
	command.SetFlagErrorFunc(usageFlagError)
	flags := command.Flags()
	flags.StringVar(&opts.Env, "env", "", "environment, one of the platform-catalog environments (required)")
	flags.StringArrayVar(&opts.Targets, "target", nil, "target name (repeatable)")
	flags.StringArrayVar(&opts.Consumers, "consumer", nil, "consumer (provision-* repository) name (repeatable)")
	flags.StringArrayVar(&opts.Declarations, "declaration", nil, "path of a shared-root.yaml (repeatable; default ./shared-root.yaml when it exists)")
	flags.BoolVar(&opts.AutomationKey, "automation-key", false, "generate the automation key pair (first controller of the environment)")
	flags.StringVar(&importKey, "automation-key-import", "", "read the automation private key from standard input: -")
	flags.StringVar(&configDir, "config-dir", "", "shared root (default ASSEMBLEIARR_CONFIG_DIR, else ~/.config/assembleiarr)")
	flags.BoolVar(&opts.DryRun, "dry-run", false, "print every action and change nothing")
	return command
}
