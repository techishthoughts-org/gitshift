package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gitshift",
	Short: "🎭 SSH-first GitHub account management",
	Long: `gitshift provides SSH-first GitHub identity management with complete isolation.

Features:
- 🔐 SSH-first approach - no GitHub API dependencies
- 🔄 Complete account isolation using SSH config
- 🔑 SSH key discovery from ~/.ssh directory
- ⚡ Fast account switching with proper isolation
- 📧 Email extraction from SSH keys
- 🛡️ No key conflicts or cross-contamination`,
	Run: func(cmd *cobra.Command, args []string) {
		// Check if version flag is set
		if version, _ := cmd.Flags().GetBool("version"); version {
			showVersion()
			return
		}

		// Show help if no command specified
		if err := cmd.Help(); err != nil {
			log.Printf("Error showing help: %v", err)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
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

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

// showVersion displays version information
func showVersion() {
	fmt.Println("🎭 gitshift - Revolutionary GitHub Identity Management")
	fmt.Println("Version: v0.1.0")
	fmt.Println("Go Version: go1.23.0")
	fmt.Println("Build Time: 2025-01-02")
	fmt.Println()
	fmt.Println("🚀 Features:")
	fmt.Println("  • Automatic GitHub account setup")
	fmt.Println("  • Smart account switching")
	fmt.Println("  • SSH key management")
	fmt.Println("  • Project-based configuration")
	fmt.Println("  • Beautiful TUI interface")
	fmt.Println()
	fmt.Println("📚 Documentation: https://github.com/techishthoughts/gitshift")
}
