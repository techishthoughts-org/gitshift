package cmd

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/gitshift/internal/discovery"
	"github.com/techishthoughts/gitshift/internal/models"
)

// planDiscoveryAliases resolves command-line selections first, then prompts
// only for results not selected explicitly. It returns a plan without changing
// discovered results or configuration state.
func planDiscoveryAliases(
	cmd *cobra.Command,
	discovered []*discovery.DiscoveredAccount,
	specs []string,
	prompt bool,
	overwrite bool,
	existingAccounts []*models.Account,
	existingPending []*models.PendingAccount,
) ([]string, error) {
	return planDiscoveryAliasesWithReader(cmd, discovered, specs, prompt, overwrite, existingAccounts, existingPending, nil)
}

func planDiscoveryAliasesWithReader(
	cmd *cobra.Command,
	discovered []*discovery.DiscoveredAccount,
	specs []string,
	prompt bool,
	overwrite bool,
	existingAccounts []*models.Account,
	existingPending []*models.PendingAccount,
	reader *bufio.Reader,
) ([]string, error) {
	aliases := make([]string, len(discovered))
	selected := make([]bool, len(discovered))
	identityIndexes := make(map[string][]int)
	suggestedIndexes := make(map[string][]int)

	for i, account := range discovered {
		if account == nil || account.Account == nil {
			return nil, fmt.Errorf("discovery result %d has no account", i+1)
		}
		aliases[i] = account.Alias
		if account.IdentityKey != "" {
			identityIndexes[account.IdentityKey] = append(identityIndexes[account.IdentityKey], i)
		}
		suggestedIndexes[account.Alias] = append(suggestedIndexes[account.Alias], i)
	}

	seenSelectors := make(map[string]struct{}, len(specs))
	for _, spec := range specs {
		selector, alias, ok := strings.Cut(spec, "=")
		if !ok || selector == "" || alias == "" {
			return nil, fmt.Errorf("invalid --alias %q: expected SELECTOR=ALIAS", spec)
		}
		if _, repeated := seenSelectors[selector]; repeated {
			return nil, fmt.Errorf("selector %q is repeated", selector)
		}
		seenSelectors[selector] = struct{}{}

		var index int
		identityMatches := identityIndexes[selector]
		switch len(identityMatches) {
		case 1:
			index = identityMatches[0]
		case 0:
			matches := suggestedIndexes[selector]
			switch len(matches) {
			case 0:
				return nil, fmt.Errorf("unknown discovery selector %q", selector)
			case 1:
				index = matches[0]
			default:
				return nil, fmt.Errorf("discovery selector %q is ambiguous; use an identity key", selector)
			}
		default:
			return nil, fmt.Errorf("discovery identity selector %q is ambiguous", selector)
		}
		if selected[index] {
			return nil, fmt.Errorf("discovery result %q was selected more than once", selector)
		}
		if err := models.ValidateAlias(alias); err != nil {
			return nil, fmt.Errorf("invalid alias %q for selector %q: %w", alias, selector, err)
		}
		aliases[index] = alias
		selected[index] = true
	}

	if prompt {
		if reader == nil {
			reader = bufio.NewReader(cmd.InOrStdin())
		}
		for i, account := range discovered {
			if selected[i] {
				continue
			}
			selector := account.IdentityKey
			if selector == "" {
				selector = account.Alias
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Alias for %s [%s]: ", selector, aliases[i]); err != nil {
				return nil, fmt.Errorf("write alias prompt: %w", err)
			}
			response, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return nil, fmt.Errorf("read alias for %q: unexpected end of input", selector)
				}
				return nil, fmt.Errorf("read alias for %q: %w", selector, err)
			}
			response = strings.TrimSpace(response)
			if response != "" {
				if err := models.ValidateAlias(response); err != nil {
					return nil, fmt.Errorf("invalid alias %q for selector %q: %w", response, selector, err)
				}
				aliases[i] = response
			}
		}
	}

	seenAliases := make(map[string]string, len(aliases)+len(existingAccounts)+len(existingPending))
	if !overwrite {
		for _, account := range existingAccounts {
			if account != nil {
				key := strings.ToLower(account.Alias)
				if previous, exists := seenAliases[key]; exists {
					return nil, fmt.Errorf("existing active aliases %q and %q collide case-insensitively", previous, account.Alias)
				}
				seenAliases[key] = account.Alias
			}
		}
		for _, account := range existingPending {
			if account != nil {
				key := strings.ToLower(account.Alias)
				if previous, exists := seenAliases[key]; exists {
					return nil, fmt.Errorf("existing aliases %q and %q collide case-insensitively", previous, account.Alias)
				}
				seenAliases[key] = account.Alias
			}
		}
	}
	for i, alias := range aliases {
		if err := models.ValidateAlias(alias); err != nil {
			return nil, fmt.Errorf("invalid alias %q for discovery result %d: %w", alias, i+1, err)
		}
		key := strings.ToLower(alias)
		if previous, exists := seenAliases[key]; exists {
			return nil, fmt.Errorf("alias %q collides case-insensitively with %q", alias, previous)
		}
		seenAliases[key] = alias
	}

	return aliases, nil
}
