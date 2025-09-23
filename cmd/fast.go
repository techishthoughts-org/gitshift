package cmd

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/cache"
	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/git"
)

// fastCmd provides fast, lightweight commands that bypass heavy initialization
var fastCmd = &cobra.Command{
	Use:   "fast",
	Short: "⚡ Fast, lightweight commands with minimal overhead",
	Long: `Fast, lightweight commands that bypass heavy service container initialization
for better performance.

These commands use caching and optimized code paths to provide near-instant
response times for common operations.

Examples:
  gitpersona fast status
  gitpersona fast current
  gitpersona fast detect`,
	Hidden: true, // Hide from main help until fully implemented
}

// fastStatusCmd provides ultra-fast status information
var fastStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Ultra-fast status check",
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()

		// Use cached data where possible
		cache := cache.GetGlobalCache()

		// Fast Git user check
		var gitName, gitEmail string
		if cached, found := cache.Get("git_config"); found {
			if gitConfig, ok := cached.(map[string]string); ok {
				gitName = gitConfig["name"]
				gitEmail = gitConfig["email"]
			}
		} else {
			// Quick Git config check without full manager initialization
			if nameCmd := exec.Command("git", "config", "--global", "user.name"); nameCmd != nil {
				if output, err := nameCmd.Output(); err == nil {
					gitName = strings.TrimSpace(string(output))
				}
			}
			if emailCmd := exec.Command("git", "config", "--global", "user.email"); emailCmd != nil {
				if output, err := emailCmd.Output(); err == nil {
					gitEmail = strings.TrimSpace(string(output))
				}
			}

			// Cache the results
			cache.Set("git_config", map[string]string{
				"name":  gitName,
				"email": gitEmail,
			})
		}

		// Fast current account check
		var currentAccount string
		if cached, found := cache.Get("current_account"); found {
			if account, ok := cached.(string); ok {
				currentAccount = account
			}
		} else {
			// Quick config load
			configManager := config.NewManager()
			if err := configManager.Load(); err == nil {
				if config := configManager.GetConfig(); config != nil {
					currentAccount = config.CurrentAccount
					cache.Set("current_account", currentAccount)
				}
			}
		}

		// Fast repository check
		isGitRepo := false
		if gitDirCmd := exec.Command("git", "rev-parse", "--git-dir"); gitDirCmd != nil {
			isGitRepo = gitDirCmd.Run() == nil
		}

		// Display results
		fmt.Printf("⚡ Fast Status (%.2fms)\n", float64(time.Since(start).Nanoseconds())/1e6)
		fmt.Printf("👤 Account: %s\n", currentAccount)
		fmt.Printf("🔧 Git Name: %s\n", gitName)
		fmt.Printf("📧 Git Email: %s\n", gitEmail)
		fmt.Printf("📁 Git Repo: %t\n", isGitRepo)

		return nil
	},
}

// fastCurrentCmd provides ultra-fast current account information
var fastCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Ultra-fast current account check",
	RunE: func(cmd *cobra.Command, args []string) error {
		cache := cache.GetGlobalCache()

		var currentAccount string
		if cached, found := cache.Get("current_account"); found {
			if account, ok := cached.(string); ok {
				currentAccount = account
			}
		} else {
			configManager := config.NewManager()
			if err := configManager.Load(); err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			if config := configManager.GetConfig(); config != nil {
				currentAccount = config.CurrentAccount
				cache.Set("current_account", currentAccount)
			}
		}

		if currentAccount == "" {
			fmt.Println("No current account set")
		} else {
			fmt.Printf("Current account: %s\n", currentAccount)
		}

		return nil
	},
}

// fastDetectCmd provides ultra-fast account detection
var fastDetectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Ultra-fast account detection",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Quick repository check
		gitManager := git.NewManager()
		if !gitManager.IsGitRepository() {
			fmt.Println("Not in a Git repository")
			return nil
		}

		// Quick remote URL check
		remoteURL, err := gitManager.GetCurrentRemoteURL("origin")
		if err != nil {
			fmt.Printf("No remote URL found: %v\n", err)
			return nil
		}

		fmt.Printf("Remote: %s\n", remoteURL)

		// Simple username extraction
		if strings.Contains(remoteURL, "github.com") {
			if strings.HasPrefix(remoteURL, "git@github.com:") {
				// SSH format: git@github.com:user/repo.git
				path := strings.TrimPrefix(remoteURL, "git@github.com:")
				parts := strings.Split(path, "/")
				if len(parts) > 0 {
					username := parts[0]
					fmt.Printf("Detected username: %s\n", username)
				}
			} else if strings.HasPrefix(remoteURL, "https://github.com/") {
				// HTTPS format: https://github.com/user/repo.git
				path := strings.TrimPrefix(remoteURL, "https://github.com/")
				parts := strings.Split(path, "/")
				if len(parts) > 0 {
					username := parts[0]
					fmt.Printf("Detected username: %s\n", username)
				}
			}
		}

		return nil
	},
}

// fastClearCacheCmd clears the performance cache
var fastClearCacheCmd = &cobra.Command{
	Use:   "clear-cache",
	Short: "Clear performance cache",
	RunE: func(cmd *cobra.Command, args []string) error {
		cache := cache.GetGlobalCache()
		size := cache.Size()
		cache.Clear()
		fmt.Printf("Cleared %d cached items\n", size)
		return nil
	},
}

// fastCacheInfoCmd shows cache information
var fastCacheInfoCmd = &cobra.Command{
	Use:   "cache-info",
	Short: "Show cache information",
	RunE: func(cmd *cobra.Command, args []string) error {
		cache := cache.GetGlobalCache()
		size := cache.Size()
		keys := cache.Keys()

		fmt.Printf("Cache size: %d items\n", size)
		if len(keys) > 0 {
			fmt.Println("Cached keys:")
			for _, key := range keys {
				fmt.Printf("  - %s\n", key)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(fastCmd)
	fastCmd.AddCommand(fastStatusCmd)
	fastCmd.AddCommand(fastCurrentCmd)
	fastCmd.AddCommand(fastDetectCmd)
	fastCmd.AddCommand(fastClearCacheCmd)
	fastCmd.AddCommand(fastCacheInfoCmd)
}
