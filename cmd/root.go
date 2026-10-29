package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	gitshiftconfig "github.com/techishthoughts/gitshift/internal/config"
)

var (
	cfgFile         string
	version         = "dev"
	buildTime       = "unknown"
	launchDefaultUI = func(cmd *cobra.Command) error { return runUI(cmd, false) }
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gitshift",
	Short: "🎭 Manage GitHub and GitLab identities with SSH",
	Long: `gitshift manages local Git identities for GitHub and GitLab accounts.

Supported Platforms:
- GitHub (github.com and GitHub Enterprise)
- GitLab (gitlab.com and self-hosted)

Features:
- 🔐 SSH-first account configuration
- 🔄 Git, SSH, GPG, and GitHub CLI switching
- 📁 Opt-in directory-based switching for zsh
- 🔑 SSH key discovery from ~/.ssh directory
- 📧 Identity discovery from SSH and GPG metadata
- 🌐 Public and self-hosted GitHub/GitLab domains`,
	Args: cobra.NoArgs,
	RunE: runRootCommand,
}

func runRootCommand(cmd *cobra.Command, _ []string) error {
	// Check if version flag is set
	if version, _ := cmd.Flags().GetBool("version"); version {
		showVersion()
		return nil
	}

	return launchDefaultUI(cmd)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func newConfigManager() *gitshiftconfig.Manager {
	if cfgFile != "" {
		return gitshiftconfig.NewManagerWithPath(cfgFile)
	}
	return gitshiftconfig.NewManager()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/gitshift/config.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("version", "v", false, "Print version information")

	// Validation commands will register themselves via their init() functions
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name "config" (without extension).
		viper.AddConfigPath(home + "/.config/gitshift")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// Read the config silently: the shell hook and prompt run this on every
	// directory change.
	_ = viper.ReadInConfig()
}

// showVersion displays version information
func showVersion() {
	fmt.Printf("gitshift %s\n", version)
	fmt.Printf("built %s with %s\n", buildTime, runtime.Version())
}
