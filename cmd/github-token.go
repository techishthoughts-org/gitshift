package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/observability"
	"github.com/techishthoughts/GitPersona/internal/services"
	"golang.org/x/term"
)

var githubTokenCmd = &cobra.Command{
	Use:   "github-token",
	Short: "Manage GitHub tokens for GitPersona accounts",
	Long: `Manage GitHub tokens for GitPersona accounts.

This command allows you to store, retrieve, and manage GitHub tokens
for different accounts without relying on external tools like GitHub CLI.`,
}

var setTokenCmd = &cobra.Command{
	Use:   "set [account]",
	Short: "Set a GitHub token for an account",
	Long: `Set a GitHub token for a specific account.

If no account is specified, the token will be set for the current account.
The token will be securely encrypted and stored locally.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSetToken,
}

var getTokenCmd = &cobra.Command{
	Use:   "get [account]",
	Short: "Get a GitHub token for an account",
	Long: `Get a GitHub token for a specific account.

If no account is specified, the token for the current account will be retrieved.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runGetToken,
}

var listTokensCmd = &cobra.Command{
	Use:   "list",
	Short: "List accounts with stored tokens",
	Long:  `List all accounts that have GitHub tokens stored.`,
	RunE:  runListTokens,
}

var deleteTokenCmd = &cobra.Command{
	Use:   "delete <account>",
	Short: "Delete a stored GitHub token",
	Long:  `Delete a stored GitHub token for the specified account.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runDeleteToken,
}

var validateTokenCmd = &cobra.Command{
	Use:   "validate [account]",
	Short: "Validate a stored GitHub token",
	Long: `Validate a stored GitHub token by making a test API call.

If no account is specified, the current account's token will be validated.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runValidateToken,
}

var importFromCLICmd = &cobra.Command{
	Use:   "import-from-cli [account]",
	Short: "Import GitHub token from GitHub CLI",
	Long: `Import the current GitHub token from GitHub CLI.

This is useful for migrating from GitHub CLI-based authentication
to GitPersona's built-in token management.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runImportFromCLI,
}

func init() {
	rootCmd.AddCommand(githubTokenCmd)

	githubTokenCmd.AddCommand(setTokenCmd)
	githubTokenCmd.AddCommand(getTokenCmd)
	githubTokenCmd.AddCommand(listTokensCmd)
	githubTokenCmd.AddCommand(deleteTokenCmd)
	githubTokenCmd.AddCommand(validateTokenCmd)
	githubTokenCmd.AddCommand(importFromCLICmd)

	// Flags for set command
	setTokenCmd.Flags().BoolP("interactive", "i", false, "Prompt for token interactively")
	setTokenCmd.Flags().StringP("token", "t", "", "GitHub token to store")
	setTokenCmd.Flags().BoolP("from-env", "e", false, "Read token from GITHUB_TOKEN environment variable")

	// Flags for get command
	getTokenCmd.Flags().BoolP("show", "s", false, "Show the actual token value (security risk)")
	getTokenCmd.Flags().BoolP("export", "x", false, "Output as export statement")
}

func runSetToken(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := observability.NewDefaultLogger()

	// Get account name
	account := "default"
	if len(args) > 0 {
		account = args[0]
	} else {
		// Try to get current account
		configManager := config.NewManager()
		if err := configManager.Load(); err == nil {
			if currentAccount, err := configManager.GetCurrentAccount(); err == nil {
				account = currentAccount.Alias
			}
		}
	}

	// Initialize token storage
	tokenStorage, err := services.NewTokenStorageService(logger)
	if err != nil {
		return fmt.Errorf("failed to initialize token storage: %w", err)
	}

	// Get token from various sources
	var token string
	if cmd.Flags().Changed("token") {
		token, _ = cmd.Flags().GetString("token")
	} else if fromEnv, _ := cmd.Flags().GetBool("from-env"); fromEnv {
		token = os.Getenv("GITHUB_TOKEN")
		if token == "" {
			return fmt.Errorf("GITHUB_TOKEN environment variable is not set")
		}
	} else if interactive, _ := cmd.Flags().GetBool("interactive"); interactive {
		token, err = promptForToken()
		if err != nil {
			return fmt.Errorf("failed to get token interactively: %w", err)
		}
	} else {
		return fmt.Errorf("no token source specified. Use --token, --from-env, or --interactive")
	}

	if token == "" {
		return fmt.Errorf("empty token provided")
	}

	// Validate token format
	if !isValidGitHubTokenFormat(token) {
		fmt.Printf("⚠️  Warning: Token doesn't match expected GitHub token format\n")
	}

	// Store token
	if err := tokenStorage.StoreToken(ctx, account, token); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	fmt.Printf("✅ GitHub token stored successfully for account: %s\n", account)
	return nil
}

func runGetToken(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := observability.NewDefaultLogger()

	// Get account name
	account := "default"
	if len(args) > 0 {
		account = args[0]
	} else {
		// Try to get current account
		configManager := config.NewManager()
		if err := configManager.Load(); err == nil {
			if currentAccount, err := configManager.GetCurrentAccount(); err == nil {
				account = currentAccount.Alias
			}
		}
	}

	// Initialize token storage
	tokenStorage, err := services.NewTokenStorageService(logger)
	if err != nil {
		return fmt.Errorf("failed to initialize token storage: %w", err)
	}

	// Get token
	token, err := tokenStorage.GetToken(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to get token for account %s: %w", account, err)
	}

	// Output format
	show, _ := cmd.Flags().GetBool("show")
	export, _ := cmd.Flags().GetBool("export")

	if export {
		fmt.Printf("export GITHUB_TOKEN=\"%s\"\n", token)
	} else if show {
		fmt.Printf("Token for account %s: %s\n", account, token)
	} else {
		// Show masked token
		masked := maskToken(token)
		fmt.Printf("Token for account %s: %s\n", account, masked)
		fmt.Printf("Use --show flag to display full token\n")
	}

	return nil
}

func runListTokens(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := observability.NewDefaultLogger()

	// Initialize token storage
	tokenStorage, err := services.NewTokenStorageService(logger)
	if err != nil {
		return fmt.Errorf("failed to initialize token storage: %w", err)
	}

	// List tokens
	accounts, err := tokenStorage.ListTokens(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tokens: %w", err)
	}

	if len(accounts) == 0 {
		fmt.Println("No stored tokens found")
		return nil
	}

	fmt.Println("Accounts with stored GitHub tokens:")
	for _, account := range accounts {
		metadata, err := tokenStorage.GetTokenMetadata(ctx, account)
		if err != nil {
			fmt.Printf("  %s (metadata unavailable)\n", account)
			continue
		}

		fmt.Printf("  %s\n", account)
		fmt.Printf("    Token prefix: %s\n", metadata.TokenPrefix)
		fmt.Printf("    Created: %s\n", metadata.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("    Last used: %s\n", metadata.LastUsed.Format("2006-01-02 15:04:05"))
	}

	return nil
}

func runDeleteToken(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := observability.NewDefaultLogger()

	account := args[0]

	// Initialize token storage
	tokenStorage, err := services.NewTokenStorageService(logger)
	if err != nil {
		return fmt.Errorf("failed to initialize token storage: %w", err)
	}

	// Confirm deletion
	fmt.Printf("Are you sure you want to delete the GitHub token for account '%s'? (y/N): ", account)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("Token deletion cancelled")
		return nil
	}

	// Delete token
	if err := tokenStorage.DeleteToken(ctx, account); err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}

	fmt.Printf("✅ GitHub token deleted successfully for account: %s\n", account)
	return nil
}

func runValidateToken(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := observability.NewDefaultLogger()

	// Get account name
	account := "default"
	if len(args) > 0 {
		account = args[0]
	} else {
		// Try to get current account
		configManager := config.NewManager()
		if err := configManager.Load(); err == nil {
			if currentAccount, err := configManager.GetCurrentAccount(); err == nil {
				account = currentAccount.Alias
			}
		}
	}

	// Initialize token storage
	tokenStorage, err := services.NewTokenStorageService(logger)
	if err != nil {
		return fmt.Errorf("failed to initialize token storage: %w", err)
	}

	// Get token
	token, err := tokenStorage.GetToken(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to get token for account %s: %w", account, err)
	}

	// Validate token
	result, err := tokenStorage.ValidateToken(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to validate token: %w", err)
	}

	if result.Valid {
		fmt.Printf("✅ Token for account '%s' is valid\n", account)
		if result.Username != "" {
			fmt.Printf("   Username: %s\n", result.Username)
		}
		if len(result.Scopes) > 0 {
			fmt.Printf("   Scopes: %s\n", strings.Join(result.Scopes, ", "))
		}
	} else {
		fmt.Printf("❌ Token for account '%s' is invalid\n", account)
		if result.Error != "" {
			fmt.Printf("   Error: %s\n", result.Error)
		}
	}

	return nil
}

func runImportFromCLI(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := observability.NewDefaultLogger()

	// Get account name
	account := "default"
	if len(args) > 0 {
		account = args[0]
	} else {
		// Try to get current account
		configManager := config.NewManager()
		if err := configManager.Load(); err == nil {
			if currentAccount, err := configManager.GetCurrentAccount(); err == nil {
				account = currentAccount.Alias
			}
		}
	}

	// Initialize services
	tokenStorage, err := services.NewTokenStorageService(logger)
	if err != nil {
		return fmt.Errorf("failed to initialize token storage: %w", err)
	}

	tokenService := services.NewGitHubTokenService(logger, nil)

	// Get token from CLI
	fmt.Println("Importing GitHub token from GitHub CLI...")
	token, err := tokenService.(*services.RealGitHubTokenService).GetCurrentGitHubToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get token from GitHub CLI: %w", err)
	}

	// Store token
	if err := tokenStorage.StoreToken(ctx, account, token); err != nil {
		return fmt.Errorf("failed to store imported token: %w", err)
	}

	fmt.Printf("✅ GitHub token imported successfully for account: %s\n", account)
	fmt.Printf("   Token prefix: %s\n", maskToken(token))

	return nil
}

// Helper functions

func promptForToken() (string, error) {
	fmt.Print("Enter GitHub token: ")
	byteToken, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	fmt.Println() // Add newline after password input

	return strings.TrimSpace(string(byteToken)), nil
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return strings.Repeat("*", len(token))
	}
	return token[:4] + strings.Repeat("*", len(token)-8) + token[len(token)-4:]
}

func isValidGitHubTokenFormat(token string) bool {
	validPrefixes := []string{"ghp_", "gho_", "ghu_", "ghs_", "ghr_"}

	for _, prefix := range validPrefixes {
		if len(token) > len(prefix) && strings.HasPrefix(token, prefix) {
			return true
		}
	}

	return false
}
