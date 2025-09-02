package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/thukabjj/GitPersona/internal/config"
	"github.com/thukabjj/GitPersona/internal/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gitpersona",
	Short: "🎭 Revolutionary TUI for seamless GitHub identity management",
	Long: `GitPersona is a revolutionary Terminal User Interface (TUI) application
that provides seamless GitHub identity management with enterprise automation
and beautiful design.

Features:
- 🚀 One-command GitHub account setup with automatic SSH key generation
- 🎨 Beautiful TUI with modern design and animations
- 🔄 Instant global account switching
- 📁 Automatic project-based identity detection
- 🔐 Enterprise-grade security with Ed25519 keys
- 🔍 Smart discovery of existing Git configurations
- 📊 Repository management and insights`,
	Run: func(cmd *cobra.Command, args []string) {
		// If no command is specified, show the TUI
		runTUI()
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/gitpersona/config.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("version", "v", false, "Print version information")
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
		viper.AddConfigPath(home + "/.config/gh-switcher")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func runTUI() {
	// Check if this is first run (no accounts configured)
	if isFirstRun() {
		fmt.Println("👋 Welcome to GitHub Account Switcher!")
		fmt.Println("🔍 It looks like this is your first time running gh-switcher.")
		fmt.Println()

		fmt.Print("Would you like to automatically discover existing Git accounts? [Y/n]: ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)

		if response == "" || strings.ToLower(response) == "y" || strings.ToLower(response) == "yes" {
			fmt.Println()
			// Run discovery
			discoverCmd.RunE(discoverCmd, []string{})
			fmt.Println()
			fmt.Println("🚀 Starting TUI...")
			fmt.Println()
		}
	}

	if err := tui.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

// isFirstRun checks if this is the first time running the application
func isFirstRun() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	configFile := filepath.Join(homeDir, ".config", "gh-switcher", "config.yaml")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return true
	}

	// Check if config exists but has no accounts
	configManager := config.NewManager()
	if err := configManager.Load(); err != nil {
		return true
	}

	accounts := configManager.ListAccounts()
	return len(accounts) == 0
}
