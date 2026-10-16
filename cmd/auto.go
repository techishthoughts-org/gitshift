package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/gitshift/internal/autoswitch"
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
		}
		return nil
	},
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
		match, err := autoswitch.Resolve(manager.ListAutoSwitchRules(), path)
		if err != nil {
			return fmt.Errorf("resolve auto-switch rule: %w", err)
		}
		if match == nil {
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
		match, err = autoswitch.Resolve(manager.ListAutoSwitchRules(), path)
		if err != nil {
			return fmt.Errorf("resolve auto-switch rule: %w", err)
		}
		if match == nil {
			return nil
		}
		if manager.GetConfig().CurrentAccount == match.Account {
			return nil
		}
		if err := manager.Load(); err != nil {
			return fmt.Errorf("load configuration: %w", err)
		}
		match, err = autoswitch.Resolve(manager.ListAutoSwitchRules(), path)
		if err != nil {
			return fmt.Errorf("resolve auto-switch rule: %w", err)
		}
		if match == nil || manager.GetConfig().CurrentAccount == match.Account {
			return nil
		}
		// Automatic switching deliberately has no force mode: any failed
		// validation aborts and rolls back through the shared switch path.
		return switchAccount(cmd, manager, match.Account, false, false, quiet)
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
	autoCmd.AddCommand(autoAddCmd, autoListCmd, autoRemoveCmd, autoResolveCmd)
	rootCmd.AddCommand(autoCmd, autoSwitchCmd)
}
