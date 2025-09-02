package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/git"
	"github.com/techishthoughts/GitPersona/internal/models"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add [alias]",
	Short: "Add a new GitHub account",
	Long: `Add a new GitHub account to the configuration.

REQUIRED FIELDS: alias, name, email, github-username
The command automatically detects if all required information is provided via flags
and runs in non-interactive mode. If any required field is missing, it will prompt
interactively unless --non-interactive is specified.

Examples:
  gitpersona add work --name "Work User" --email "work@company.com" --github-username "workuser"
  gitpersona add personal --name "Personal User" --email "user@example.com" --github-username "username"
  gitpersona add work --name "Work" --email "work@company.com" --github-username "workuser" --ssh-key "~/.ssh/id_rsa_work"
  gitpersona add work --non-interactive  # will fail if required fields missing

💡 TIP: Use 'gitpersona add-github username' for automatic setup from GitHub API!`,
	Args: cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configManager := config.NewManager()
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
		sshKey, _ := cmd.Flags().GetString("ssh-key")
		description, _ := cmd.Flags().GetString("description")
		setDefault, _ := cmd.Flags().GetBool("default")
		nonInteractive, _ := cmd.Flags().GetBool("non-interactive")

		// Check if all required fields are provided via flags
		allRequiredProvided := (len(args) > 0 || alias != "") && name != "" && email != "" && githubUsername != ""

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

		if githubUsername == "" {
			if useInteractiveMode {
				githubUsername = promptForInput("GitHub username (without @): ")
			} else {
				return fmt.Errorf("GitHub username is required. Use --github-username flag or --non-interactive=false for interactive mode")
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
			if err := gitManager.ValidateSSHKey(sshKey); err != nil {
				fmt.Printf("⚠️  Warning: SSH key validation failed: %v\n", err)
				fmt.Println("   Account will be created, but SSH key may not work properly.")
				if useInteractiveMode {
					fmt.Print("Continue anyway? [y/N]: ")
					confirmation := promptForInput("")
					if confirmation != "y" && confirmation != "Y" && confirmation != "yes" && confirmation != "Yes" {
						return fmt.Errorf("account creation cancelled")
					}
				}
			}
		}

		// Clean GitHub username (remove @ if provided)
		githubUsername = strings.TrimPrefix(githubUsername, "@")

		// Create the account
		account := models.NewAccount(alias, name, email, sshKey)
		account.GitHubUsername = githubUsername
		account.Description = description

		// Add the account to configuration
		if err := configManager.AddAccount(account); err != nil {
			return fmt.Errorf("failed to add account: %w", err)
		}

		// Set as default if requested or if it's the first account
		if setDefault || len(configManager.GetConfig().Accounts) == 1 {
			if err := configManager.SetCurrentAccount(alias); err != nil {
				return fmt.Errorf("failed to set as current account: %w", err)
			}
		}

		fmt.Printf("✅ Successfully added account '%s'\n", alias)
		fmt.Printf("   Name: %s\n", name)
		fmt.Printf("   Email: %s\n", email)
		fmt.Printf("   GitHub: @%s\n", githubUsername)
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
	addCmd.Flags().StringP("ssh-key", "k", "", "Path to SSH private key (optional)")
	addCmd.Flags().StringP("description", "d", "", "Account description (optional)")
	addCmd.Flags().BoolP("default", "", false, "Set as default account")
	addCmd.Flags().Bool("non-interactive", false, "Run in non-interactive mode (no prompts)")
}

// promptForInput prompts the user for input and returns the trimmed response
func promptForInput(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}
