package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/gitshift/internal/autoswitch"
	"github.com/techishthoughts/gitshift/internal/config"
)

var autoCmd = &cobra.Command{
	Use:   "auto",
	Short: "Manage directory-to-account rules",
}

var autoAddCmd = &cobra.Command{
	Use:   "add PATH ALIAS",
	Short: "Add or replace an auto-switch rule",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		manager := newConfigManager()
		if err := manager.Load(); err != nil {
			return fmt.Errorf("load configuration: %w", err)
		}
		rule, err := manager.AddAutoSwitchRule(args[0], args[1])
		if err != nil {
			return fmt.Errorf("add auto-switch rule: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Auto-switch rule: %s -> %s\n", rule.Path, rule.Account)
		return nil
	},
}

var autoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List auto-switch rules",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		manager := newConfigManager()
		if err := manager.LoadReadOnly(); err != nil {
			return fmt.Errorf("load configuration: %w", err)
		}
		rules := manager.ListAutoSwitchRules()
		asJSON, _ := cmd.Flags().GetBool("json")
		if asJSON {
			return writeJSON(cmd, rules)
		}
		if len(rules) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No auto-switch rules configured.")
			return nil
		}
		for _, rule := range rules {
			fmt.Fprintf(cmd.OutOrStdout(), "%s -> %s\n", rule.Path, rule.Account)
		}
		return nil
	},
}

var autoRemoveCmd = &cobra.Command{
	Use:   "remove PATH",
	Short: "Remove an auto-switch rule",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		manager := newConfigManager()
		if err := manager.Load(); err != nil {
			return fmt.Errorf("load configuration: %w", err)
		}
		if err := manager.RemoveAutoSwitchRule(args[0]); err != nil {
			return fmt.Errorf("remove auto-switch rule: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Auto-switch rule removed.")
		return nil
	},
}

var autoResolveCmd = &cobra.Command{
	Use:   "resolve [PATH]",
	Short: "Resolve the account for a directory without switching",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := pathArgument(args)
		if err != nil {
			return err
		}
		manager := newConfigManager()
		if err := manager.LoadReadOnly(); err != nil {
			return fmt.Errorf("load configuration: %w", err)
		}
		match, err := autoswitch.Resolve(manager.ListAutoSwitchRules(), path)
		if err != nil {
			return fmt.Errorf("resolve auto-switch rule: %w", err)
		}
		asJSON, _ := cmd.Flags().GetBool("json")
		if asJSON {
			return writeJSON(cmd, match)
		}
		if match != nil {
			fmt.Fprintln(cmd.OutOrStdout(), match.Account)
		} else if fallback := manager.GetConfig().AutoSwitch.DefaultAccount; fallback != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "%s (default)\n", fallback)
		}
		return nil
	},
}

var autoDefaultCmd = &cobra.Command{
	Use:   "default [ALIAS]",
	Short: "Show or set the account used when no rule matches",
	Long: `Show or set the account auto-switch uses outside every rule directory.

Without a default, leaving rule directories keeps the current identity.`,
	Example: `  gitshift auto default work
  gitshift auto default
  gitshift auto default --clear`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		clear, _ := cmd.Flags().GetBool("clear")
		if clear && len(args) > 0 {
			return fmt.Errorf("use either ALIAS or --clear, not both")
		}
		manager := newConfigManager()
		if !clear && len(args) == 0 {
			if err := manager.LoadReadOnly(); err != nil {
				return fmt.Errorf("load configuration: %w", err)
			}
			if fallback := manager.GetConfig().AutoSwitch.DefaultAccount; fallback != "" {
				fmt.Fprintln(cmd.OutOrStdout(), fallback)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "No auto-switch default account configured.")
			}
			return nil
		}
		if err := manager.Load(); err != nil {
			return fmt.Errorf("load configuration: %w", err)
		}
		alias := ""
		if len(args) == 1 {
			alias = args[0]
		}
		if err := manager.SetAutoSwitchDefault(alias); err != nil {
			return fmt.Errorf("set auto-switch default: %w", err)
		}
		if alias == "" {
			fmt.Fprintln(cmd.OutOrStdout(), "Auto-switch default account cleared.")
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Auto-switch default account: %s\n", alias)
		}
		return nil
	},
}

// autoSwitchTarget returns the account for path: the deepest matching rule, or
// the configured default when no rule matches. Empty means leave state alone.
func autoSwitchTarget(manager *config.Manager, path string) (string, error) {
	match, err := autoswitch.Resolve(manager.ListAutoSwitchRules(), path)
	if err != nil {
		return "", fmt.Errorf("resolve auto-switch rule: %w", err)
	}
	if match != nil {
		return match.Account, nil
	}
	return manager.GetConfig().AutoSwitch.DefaultAccount, nil
}

var autoSwitchCmd = &cobra.Command{
	Use:   "auto-switch [PATH]",
	Short: "Switch to the account selected for a directory",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := pathArgument(args)
		if err != nil {
			return err
		}
		quiet, _ := cmd.Flags().GetBool("quiet")
		manager := newConfigManager()
		// Keep the common no-match path strictly read-only. Once a rule is
		// found, lock and reload before trusting it or mutating global state.
		if err := manager.LoadReadOnly(); err != nil {
			return fmt.Errorf("load configuration: %w", err)
		}
		target, err := autoSwitchTarget(manager, path)
		if err != nil {
			return err
		}
		if target == "" || manager.GetConfig().CurrentAccount == target {
			return nil
		}
		release, err := manager.AcquireSwitchLock()
		if err != nil {
			return fmt.Errorf("acquire switch transaction: %w", err)
		}
		defer release()
		if err := manager.LoadReadOnly(); err != nil {
			return fmt.Errorf("load configuration: %w", err)
		}
		target, err = autoSwitchTarget(manager, path)
		if err != nil {
			return err
		}
		if target == "" || manager.GetConfig().CurrentAccount == target {
			return nil
		}
		if err := manager.Load(); err != nil {
			return fmt.Errorf("load configuration: %w", err)
		}
		target, err = autoSwitchTarget(manager, path)
		if err != nil {
			return err
		}
		if target == "" || manager.GetConfig().CurrentAccount == target {
			return nil
		}
		// Automatic switching deliberately has no force mode: any failed
		// validation aborts and rolls back through the shared switch path.
		// It runs on every directory change, so it skips the SSH round-trips
		// and keeps the local key and identity checks.
		return switchAccount(cmd, manager, target, false, true, quiet)
	},
}

func pathArgument(args []string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	path, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get current directory: %w", err)
	}
	return path, nil
}

func writeJSON(cmd *cobra.Command, value any) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func init() {
	autoListCmd.Flags().Bool("json", false, "output JSON")
	autoResolveCmd.Flags().Bool("json", false, "output JSON")
	autoSwitchCmd.Flags().Bool("quiet", false, "suppress successful switch output")
	autoDefaultCmd.Flags().Bool("clear", false, "remove the default account")
	autoCmd.AddCommand(autoAddCmd, autoListCmd, autoRemoveCmd, autoResolveCmd, autoDefaultCmd)
	rootCmd.AddCommand(autoCmd, autoSwitchCmd)
}
