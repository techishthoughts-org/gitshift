package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var shellInitCmd = &cobra.Command{
	Use:   "shell-init zsh",
	Short: "Print the opt-in zsh auto-switch hook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if args[0] != "zsh" {
			return fmt.Errorf("unsupported shell %q; only zsh is supported", args[0])
		}
		executable, err := os.Executable()
		if err != nil {
			return fmt.Errorf("locate gitshift executable: %w", err)
		}
		executable = hookExecutable(executable, exec.LookPath)
		configArgument := ""
		if cfgFile != "" {
			absoluteConfig, absErr := filepath.Abs(cfgFile)
			if absErr != nil {
				return fmt.Errorf("make configuration path absolute: %w", absErr)
			}
			configArgument = " --config " + shellQuote(filepath.Clean(absoluteConfig))
		}
		_, err = fmt.Fprint(cmd.OutOrStdout(), zshHook(executable, configArgument))
		return err
	},
}

// hookExecutable prefers the PATH entry that runs this same binary, such as
// Homebrew's /opt/homebrew/bin/gitshift symlink, so the hook survives upgrades
// that remove the versioned file. Otherwise it uses the physical path.
func hookExecutable(executable string, lookPath func(string) (string, error)) string {
	physical := executable
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		physical = resolved
	}
	onPath, err := lookPath(filepath.Base(executable))
	if err != nil {
		return physical
	}
	onPath, err = filepath.Abs(onPath)
	if err != nil {
		return physical
	}
	if resolved, err := filepath.EvalSymlinks(onPath); err == nil && resolved == physical {
		return filepath.Clean(onPath)
	}
	return physical
}

func zshHook(executable, configArgument string) string {
	binary := shellQuote(filepath.Clean(executable))
	return `# GitShift directory auto-switch hook
_gitshift_auto_switch() {
  [[ -o interactive ]] || return 0
  [[ "${GITSHIFT_AUTO_SWITCH_DISABLE:-}" != "1" ]] || return 0
  [[ -z "${_GITSHIFT_AUTO_SWITCH_RUNNING:-}" ]] || return 0
  [[ "${_GITSHIFT_AUTO_SWITCH_LAST_DIR:-}" != "$PWD" ]] || return 0

  _GITSHIFT_AUTO_SWITCH_RUNNING=1
  local gitshift_status
  if command ` + binary + configArgument + ` auto-switch --quiet -- "$PWD"; then
    gitshift_status=0
  else
    gitshift_status=$?
  fi
  unset _GITSHIFT_AUTO_SWITCH_RUNNING
  if (( gitshift_status == 0 )); then
    _GITSHIFT_AUTO_SWITCH_LAST_DIR="$PWD"
  fi
  return $gitshift_status
}

autoload -Uz add-zsh-hook
add-zsh-hook -d chpwd _gitshift_auto_switch 2>/dev/null
add-zsh-hook chpwd _gitshift_auto_switch
unset _GITSHIFT_AUTO_SWITCH_LAST_DIR
_gitshift_auto_switch
`
}

func init() { rootCmd.AddCommand(shellInitCmd) }
