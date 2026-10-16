package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/gitshift/internal/git"
	"github.com/techishthoughts/gitshift/internal/models"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add [alias]",
	Short: "Add a GitHub or GitLab account",
	Long: `Add a new Git platform account to the configuration.

Supported Platforms:
- GitHub (github.com and GitHub Enterprise)
- GitLab (gitlab.com and self-hosted)

REQUIRED FIELDS: alias, name, email, username
The command automatically detects if all required information is provided via flags
and runs in non-interactive mode. If any required field is missing, it will prompt
interactively unless --non-interactive is specified.

Examples:
  # GitHub account
  gitshift add work --name "Work User" --email "work@company.com" --username "workuser"
  gitshift add personal --name "Personal User" --email "user@example.com" --username "username"

  # GitLab account
  gitshift add gitlab-work --platform gitlab --name "Work User" --email "work@company.com" --username "workuser"

  # Self-hosted GitLab
  gitshift add company --platform gitlab --domain gitlab.company.example --name "Work" --email "work@company.com" --username "workuser" --ssh-key "~/.ssh/id_rsa_work"

  # GitHub Enterprise
  gitshift add enterprise --domain github.company.example --name "Enterprise User" --email "user@company.com" --username "user"

  gitshift add work --non-interactive  # will fail if required fields missing

💡 TIP: The --github-username flag works for any Git platform (naming is for backward compatibility)`,
	Args: cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configManager := newConfigManager()
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		gitManager := git.NewManager()

		var alias string
		if len(args) > 0 {
			alias = args[0]
		}

		// Get values from flags
		name, _ := cmd.Flags().GetString("name")
		email, _ := cmd.Flags().GetString("email")
		githubUsername, _ := cmd.Flags().GetString("github-username")
		username, _ := cmd.Flags().GetString("username")
		provider, _ := cmd.Flags().GetString("platform")
		domain, _ := cmd.Flags().GetString("domain")
		apiEndpoint, _ := cmd.Flags().GetString("api-endpoint")
		sshKey, _ := cmd.Flags().GetString("ssh-key")
		description, _ := cmd.Flags().GetString("description")
		setDefault, _ := cmd.Flags().GetBool("default")
		nonInteractive, _ := cmd.Flags().GetBool("non-interactive")
		enableGPG, _ := cmd.Flags().GetBool("enable-gpg")

		// Check if all required fields are provided via flags
		if username == "" {
			username = githubUsername
		}
		allRequiredProvided := (len(args) > 0 || alias != "") && name != "" && email != "" && username != ""

		// Determine if we should run in interactive mode
		useInteractiveMode := !nonInteractive && !allRequiredProvided

		// Interactive mode if values are missing and not in non-interactive mode
		if alias == "" {
			if useInteractiveMode {
				alias = promptForInput("Account alias (e.g., 'work', 'personal'): ")
			} else {
				return fmt.Errorf("account alias is required. Use --non-interactive=false for interactive mode")
			}
		}

		if name == "" {
			if useInteractiveMode {
				name = promptForInput("Git user name: ")
			} else {
				return fmt.Errorf("name is required. Use --name flag or --non-interactive=false for interactive mode")
			}
		}

		if email == "" {
			if useInteractiveMode {
				email = promptForInput("Git user email: ")
			} else {
				return fmt.Errorf("email is required. Use --email flag or --non-interactive=false for interactive mode")
			}
		}

		if username == "" {
			if useInteractiveMode {
				username = promptForInput("Platform username (without @): ")
			} else {
				return fmt.Errorf("username is required. Use --username or --github-username")
			}
		}

		// For optional fields, only prompt if in interactive mode and flag wasn't explicitly set
		sshKeyFlag := cmd.Flags().Lookup("ssh-key")
		if useInteractiveMode && (sshKeyFlag == nil || !sshKeyFlag.Changed) && sshKey == "" {
			sshKey = promptForInput("SSH key path (optional, press Enter to skip): ")
		}

		descFlag := cmd.Flags().Lookup("description")
		if useInteractiveMode && (descFlag == nil || !descFlag.Changed) && description == "" {
			description = promptForInput("Description (optional): ")
		}

		// Validate the SSH key if provided (only warn, don't fail)
		if sshKey != "" {
			var err error
			sshKey, err = models.NormalizeSSHKeyPath(sshKey)
			if err != nil {
				return err
			}
			if err := gitManager.ValidateSSHKey(sshKey); err != nil {
				return fmt.Errorf("invalid SSH key: %w", err)
			}
		}

		username = strings.TrimPrefix(username, "@")
		provider = strings.ToLower(strings.TrimSpace(provider))
		if provider == "" {
			provider = "github"
		}

		// Create the account
		account := models.NewAccount(alias, name, email, sshKey)
		account.Platform = provider
		account.Username = username
		account.Domain = domain
		account.APIEndpoint = apiEndpoint
		account.IsDefault = setDefault
		if provider == "github" {
			account.GitHubUsername = username
		}
		account.Description = description
		if err := account.Validate(); err != nil {
			return fmt.Errorf("invalid account: %w", err)
		}

		// Handle GPG signing preference
		if enableGPG {
			account.GPGEnabled = true
			fmt.Printf("🔐 GPG commit signing will be enabled for this account\n")
			fmt.Printf("   💡 Generate a GPG key with: gitshift gpg-keygen %s\n", alias)
		}

		// Add the account to configuration
		if err := configManager.AddAccount(account); err != nil {
			return fmt.Errorf("failed to add account: %w", err)
		}

		fmt.Printf("✅ Successfully added account '%s'\n", alias)
		fmt.Printf("   Name: %s\n", name)
		fmt.Printf("   Email: %s\n", email)
		fmt.Printf("   Platform: %s (%s)\n", account.GetPlatform(), account.GetDomain())
		fmt.Printf("   Username: @%s\n", username)
		if sshKey != "" {
			fmt.Printf("   SSH Key: %s\n", sshKey)
		}
		if description != "" {
			fmt.Printf("   Description: %s\n", description)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.Flags().StringP("name", "n", "", "Git user name (required)")
	addCmd.Flags().StringP("email", "e", "", "Git user email (required)")
	addCmd.Flags().StringP("github-username", "g", "", "GitHub username without @ (required)")
	addCmd.Flags().String("username", "", "Platform username without @ (required)")
	addCmd.Flags().String("platform", "github", "Git hosting platform (github or gitlab)")
	addCmd.Flags().String("domain", "", "Platform hostname (defaults from platform)")
	addCmd.Flags().String("api-endpoint", "", "HTTPS API endpoint for self-hosted platforms")
	addCmd.Flags().StringP("ssh-key", "k", "", "Path to SSH private key (optional)")
	addCmd.Flags().StringP("description", "d", "", "Account description (optional)")
	addCmd.Flags().BoolP("default", "", false, "Set as default account")
	addCmd.Flags().Bool("non-interactive", false, "Run in non-interactive mode (no prompts)")
	addCmd.Flags().Bool("enable-gpg", false, "Enable GPG commit signing for this account (requires GPG key)")
}

// promptForInput prompts the user for input and returns the trimmed response
func promptForInput(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}
