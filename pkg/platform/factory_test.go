package platform

import "testing"

func TestCreatePlatformFromConfigAppliesCustomAPIEndpoint(t *testing.T) {
	const endpoint = "https://git.example.com/custom/api"
	p, err := NewFactory().CreatePlatformFromConfig(&Config{Type: TypeGitLab, Domain: "git.example.com", APIEndpoint: endpoint})
	if err != nil {
		t.Fatal(err)
	}
	gl, ok := p.(*GitLabPlatform)
	if !ok || gl.apiEndpoint != endpoint {
		t.Fatalf("custom endpoint not applied: %#v", p)
	}
}

func TestGitHubEnterpriseDefaultsAPIEndpoint(t *testing.T) {
	p := NewGitHubEnterprisePlatform("github.example.com", "")
	if p.apiEndpoint != "https://github.example.com/api/v3" {
		t.Fatalf("unexpected endpoint %q", p.apiEndpoint)
	}
}
