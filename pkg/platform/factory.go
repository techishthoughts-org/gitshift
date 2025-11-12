package platform

import (
	"fmt"
)

// Factory creates platform instances based on configuration
type Factory struct {
	registry *Registry
}

// NewFactory creates a new platform factory with default platforms registered
func NewFactory() *Factory {
	registry := NewRegistry()

	// Register default platforms
	registry.Register(NewGitHubPlatform())
	registry.Register(NewGitLabPlatform())

	return &Factory{
		registry: registry,
	}
}

// GetPlatform returns a platform instance based on type and optional domain
func (f *Factory) GetPlatform(platformType Type, domain string) (Platform, error) {
	switch platformType {
	case TypeGitHub:
		if domain == "" || domain == "github.com" {
			return NewGitHubPlatform(), nil
		}
		// GitHub Enterprise
		return NewGitHubEnterprisePlatform(domain, ""), nil

	case TypeGitLab:
		if domain == "" || domain == "gitlab.com" {
			return NewGitLabPlatform(), nil
		}
		// Self-hosted GitLab
		return NewGitLabSelfHostedPlatform(domain, ""), nil

	case TypeBitbucket:
		// TODO: Implement Bitbucket support
		return nil, fmt.Errorf("Bitbucket platform not yet implemented")

	case TypeCustom:
		if domain == "" {
			return nil, fmt.Errorf("custom platform requires domain")
		}
		// TODO: Implement custom platform support
		return nil, fmt.Errorf("custom platform not yet implemented")

	default:
		return nil, fmt.Errorf("unsupported platform type: %s", platformType)
	}
}

// GetPlatformByDomain returns a platform instance by auto-detecting from domain
func (f *Factory) GetPlatformByDomain(domain string) (Platform, error) {
	platformType := DetectPlatformFromDomain(domain)
	return f.GetPlatform(platformType, domain)
}

// GetRegistry returns the platform registry
func (f *Factory) GetRegistry() *Registry {
	return f.registry
}

// DetectPlatformFromDomain detects the platform type from a domain
func DetectPlatformFromDomain(domain string) Type {
	switch {
	case domain == "github.com" || containsAny(domain, []string{"github"}):
		return TypeGitHub
	case domain == "gitlab.com" || containsAny(domain, []string{"gitlab"}):
		return TypeGitLab
	case domain == "bitbucket.org" || containsAny(domain, []string{"bitbucket"}):
		return TypeBitbucket
	default:
		return TypeCustom
	}
}

// CreatePlatformFromConfig creates a platform instance from configuration
func (f *Factory) CreatePlatformFromConfig(cfg *Config) (Platform, error) {
	if cfg == nil {
		return nil, fmt.Errorf("platform config cannot be nil")
	}

	if !cfg.Type.IsValid() {
		return nil, fmt.Errorf("invalid platform type: %s", cfg.Type)
	}

	platform, err := f.GetPlatform(cfg.Type, cfg.Domain)
	if err != nil {
		return nil, err
	}

	// TODO: Apply additional config like custom API endpoint, token, etc.

	return platform, nil
}

// ListSupportedPlatforms returns a list of all supported platform types
func (f *Factory) ListSupportedPlatforms() []Type {
	return []Type{TypeGitHub, TypeGitLab}
}
