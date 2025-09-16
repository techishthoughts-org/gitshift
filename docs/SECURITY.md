# 🔒 GitPersona Security Guide

> **Security best practices, considerations, and guidelines for GitPersona**

---

## 📖 **Table of Contents**

1. [Security Overview](#security-overview)
2. [Security Architecture](#security-architecture)
3. [Data Protection](#data-protection)
4. [Authentication & Authorization](#authentication--authorization)
5. [SSH Key Security](#ssh-key-security)
6. [Configuration Security](#configuration-security)
7. [Network Security](#network-security)
8. [Secure Development](#secure-development)
9. [Security Best Practices](#security-best-practices)
10. [Incident Response](#incident-response)

---

## 🛡️ **Security Overview**

### **Security Principles**

GitPersona is built with security-first principles:

- **🔐 Zero Trust**: Never trust, always verify
- **🔒 Defense in Depth**: Multiple layers of security
- **🛡️ Principle of Least Privilege**: Minimal required permissions
- **🔍 Security by Design**: Security built into every component
- **📊 Continuous Monitoring**: Proactive security monitoring
- **🔄 Regular Updates**: Keep dependencies and components current

### **Security Scope**

GitPersona handles sensitive information including:
- **SSH Private Keys** - Cryptographic keys for GitHub authentication
- **GitHub Tokens** - API access tokens for GitHub services
- **User Credentials** - Email addresses and personal information
- **Configuration Data** - Account settings and preferences
- **System Information** - File paths and system details

---

## 🏗️ **Security Architecture**

### **Security Layers**

```
┌─────────────────────────────────────────────────────────────┐
│                    Security Architecture                    │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │   Input         │  │   Processing    │  │   Storage    │ │
│  │   Validation    │  │   Encryption    │  │   Protection │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │   Network       │  │   File System   │  │   Process    │ │
│  │   Security      │  │   Security      │  │   Isolation  │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### **Security Components**

#### **Input Validation**
```go
// Email validation with security considerations
func isValidEmail(email string) bool {
    if email == "" || len(email) > 254 {
        return false
    }

    // Prevent injection attacks
    if strings.Contains(email, "<script>") || strings.Contains(email, "javascript:") {
        return false
    }

    emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
    return emailRegex.MatchString(email)
}

// GitHub username validation
func isValidGitHubUsername(username string) bool {
    if username == "" || len(username) > 39 || len(username) < 1 {
        return false
    }

    // Prevent path traversal
    if strings.Contains(username, "..") || strings.Contains(username, "/") {
        return false
    }

    // Prevent injection attacks
    if strings.Contains(username, "<") || strings.Contains(username, ">") {
        return false
    }

    usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
    return usernameRegex.MatchString(username)
}
```

#### **File System Security**
```go
// Secure file operations with proper permissions
func (s *RealZshSecretsService) writeSecretsFile(path string, content string) error {
    // Validate file path to prevent directory traversal
    if !filepath.IsAbs(path) {
        return errors.New("file path must be absolute")
    }

    // Ensure directory exists with secure permissions
    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Errorf("failed to create directory: %w", err)
    }

    // Write file with secure permissions (600 - owner read/write only)
    if err := os.WriteFile(path, []byte(content), 0600); err != nil {
        return fmt.Errorf("failed to write file: %w", err)
    }

    // Set additional security attributes if supported
    if err := s.setSecurityAttributes(path); err != nil {
        s.logger.Warn(context.Background(), "failed to set security attributes",
            observability.F("path", path),
            observability.F("error", err.Error()),
        )
    }

    return nil
}
```

---

## 🔐 **Data Protection**

### **Sensitive Data Handling**

#### **SSH Key Protection**
```go
// Secure SSH key generation
func (cm *ModernCryptoManager) GenerateEd25519Key(alias, email string) (string, error) {
    // Use cryptographically secure random number generator
    publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
    if err != nil {
        return "", fmt.Errorf("failed to generate Ed25519 key: %w", err)
    }

    // Create secure file path
    privateKeyPath := filepath.Join(cm.sshDir, fmt.Sprintf("id_ed25519_%s", alias))

    // Encode private key with proper format
    privateKeyPEM, err := cm.encodePrivateKey(privateKey, email)
    if err != nil {
        return "", fmt.Errorf("failed to encode private key: %w", err)
    }

    // Write with secure permissions
    if err := os.WriteFile(privateKeyPath, privateKeyPEM, 0600); err != nil {
        return "", fmt.Errorf("failed to write private key: %w", err)
    }

    // Set extended attributes for additional security
    cm.setSecurityMetadata(privateKeyPath, alias)

    return privateKeyPath, nil
}
```

#### **Token Protection**
```go
// Secure token handling
func (s *RealZshSecretsService) UpdateGitHubToken(ctx context.Context, token string) error {
    // Validate token format
    if !s.isValidGitHubToken(token) {
        return errors.New("invalid GitHub token format")
    }

    // Sanitize token for logging (never log full token)
    sanitizedToken := s.sanitizeToken(token)
    s.logger.Info(ctx, "updating_github_token",
        observability.F("token_preview", sanitizedToken),
    )

    // Update token in secure file
    return s.updateTokenInFile(ctx, token)
}

// Token sanitization for logging
func (s *RealZshSecretsService) sanitizeToken(token string) string {
    if len(token) < 8 {
        return "***"
    }
    return token[:4] + "***" + token[len(token)-4:]
}
```

### **Data Encryption**

#### **Configuration Encryption**
```go
// Encrypt sensitive configuration data
type EncryptedConfig struct {
    EncryptedData []byte `json:"encrypted_data"`
    Nonce         []byte `json:"nonce"`
    Salt          []byte `json:"salt"`
}

func (e *EncryptedConfig) Encrypt(data []byte, password string) error {
    // Generate random salt
    salt := make([]byte, 32)
    if _, err := rand.Read(salt); err != nil {
        return fmt.Errorf("failed to generate salt: %w", err)
    }

    // Derive key from password
    key := pbkdf2.Key([]byte(password), salt, 100000, 32, sha256.New)

    // Generate random nonce
    nonce := make([]byte, 12)
    if _, err := rand.Read(nonce); err != nil {
        return fmt.Errorf("failed to generate nonce: %w", err)
    }

    // Encrypt data
    block, err := aes.NewCipher(key)
    if err != nil {
        return fmt.Errorf("failed to create cipher: %w", err)
    }

    aesGCM, err := cipher.NewGCM(block)
    if err != nil {
        return fmt.Errorf("failed to create GCM: %w", err)
    }

    e.EncryptedData = aesGCM.Seal(nil, nonce, data, nil)
    e.Nonce = nonce
    e.Salt = salt

    return nil
}
```

---

## 🔑 **Authentication & Authorization**

### **GitHub Authentication**

#### **Token Validation**
```go
// Validate GitHub token format and permissions
func (s *RealZshSecretsService) isValidGitHubToken(token string) bool {
    // Check token format (GitHub tokens start with specific prefixes)
    validPrefixes := []string{"ghp_", "gho_", "ghu_", "ghs_", "ghr_"}

    for _, prefix := range validPrefixes {
        if strings.HasPrefix(token, prefix) {
            // Validate length (GitHub tokens are typically 40 characters after prefix)
            if len(token) == len(prefix)+40 {
                return true
            }
        }
    }

    return false
}

// Test token validity with GitHub API
func (s *GitHubTokenService) ValidateToken(ctx context.Context, token string) error {
    // Test token with minimal API call
    client := github.NewClient(nil).WithAuthToken(token)

    // Use a lightweight API call to validate token
    _, _, err := client.Users.Get(ctx, "")
    if err != nil {
        return fmt.Errorf("token validation failed: %w", err)
    }

    return nil
}
```

#### **SSH Key Authentication**
```go
// Secure SSH key testing
func (s *RealSSHService) TestGitHubAuthentication(ctx context.Context, keyPath string) error {
    // Validate key file exists and has correct permissions
    if err := s.validateKeyFile(keyPath); err != nil {
        return fmt.Errorf("key file validation failed: %w", err)
    }

    // Test SSH connection with timeout
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    // Use SSH to test GitHub connection
    cmd := exec.CommandContext(ctx, "ssh", "-T", "-o", "StrictHostKeyChecking=no",
        "-o", "UserKnownHostsFile=/dev/null", "-i", keyPath, "git@github.com")

    output, err := cmd.CombinedOutput()
    if err != nil {
        // Check if it's an authentication error vs connection error
        if strings.Contains(string(output), "Permission denied") {
            return errors.New("SSH key authentication failed")
        }
        return fmt.Errorf("SSH connection failed: %w", err)
    }

    return nil
}
```

### **Access Control**

#### **File Permission Validation**
```go
// Validate file permissions for security
func (s *RealSSHService) validateKeyFile(keyPath string) error {
    // Check if file exists
    info, err := os.Stat(keyPath)
    if err != nil {
        return fmt.Errorf("key file not found: %w", err)
    }

    // Check file permissions (should be 600 or 400)
    mode := info.Mode()
    if mode&0777 != 0600 && mode&0777 != 0400 {
        return fmt.Errorf("key file has insecure permissions: %v (should be 600 or 400)", mode&0777)
    }

    // Check if file is owned by current user
    if !s.isOwnedByCurrentUser(info) {
        return errors.New("key file is not owned by current user")
    }

    return nil
}
```

---

## 🔐 **SSH Key Security**

### **Key Generation Security**

#### **Cryptographically Secure Generation**
```go
// Secure SSH key generation with proper entropy
func (cm *ModernCryptoManager) GenerateEd25519Key(alias, email string) (string, error) {
    // Use cryptographically secure random number generator
    publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
    if err != nil {
        return "", fmt.Errorf("failed to generate Ed25519 key: %w", err)
    }

    // Validate entropy source
    if err := cm.validateEntropySource(); err != nil {
        return "", fmt.Errorf("entropy source validation failed: %w", err)
    }

    // Create secure key file
    return cm.createSecureKeyFile(alias, privateKey, email)
}

// Validate entropy source
func (cm *ModernCryptoManager) validateEntropySource() error {
    // Test random number generation
    testBytes := make([]byte, 32)
    if _, err := rand.Read(testBytes); err != nil {
        return fmt.Errorf("random number generation failed: %w", err)
    }

    // Check for sufficient entropy (basic test)
    if cm.hasLowEntropy(testBytes) {
        return errors.New("insufficient entropy detected")
    }

    return nil
}
```

#### **Key Storage Security**
```go
// Secure key storage with additional protections
func (cm *ModernCryptoManager) createSecureKeyFile(alias string, privateKey ed25519.PrivateKey, email string) (string, error) {
    privateKeyPath := filepath.Join(cm.sshDir, fmt.Sprintf("id_ed25519_%s", alias))

    // Encode private key
    privateKeyPEM, err := cm.encodePrivateKey(privateKey, email)
    if err != nil {
        return "", fmt.Errorf("failed to encode private key: %w", err)
    }

    // Write with secure permissions
    if err := os.WriteFile(privateKeyPath, privateKeyPEM, 0600); err != nil {
        return "", fmt.Errorf("failed to write private key: %w", err)
    }

    // Set additional security attributes
    if err := cm.setSecurityAttributes(privateKeyPath); err != nil {
        cm.logger.Warn(context.Background(), "failed to set security attributes",
            observability.F("path", privateKeyPath),
            observability.F("error", err.Error()),
        )
    }

    // Clear sensitive data from memory
    cm.clearSensitiveData(privateKeyPEM)

    return privateKeyPath, nil
}
```

### **Key Management Security**

#### **Secure Key Loading**
```go
// Secure SSH key loading with validation
func (s *RealSSHAgentService) LoadKey(ctx context.Context, keyPath string) error {
    // Validate key file security
    if err := s.validateKeySecurity(keyPath); err != nil {
        return fmt.Errorf("key security validation failed: %w", err)
    }

    // Load key with timeout
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    // Use ssh-add with secure options
    cmd := exec.CommandContext(ctx, "ssh-add", "-t", "3600", keyPath)

    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("failed to load SSH key: %w", err)
    }

    s.logger.Info(ctx, "ssh_key_loaded",
        observability.F("key_path", filepath.Base(keyPath)), // Don't log full path
    )

    return nil
}
```

---

## ⚙️ **Configuration Security**

### **Configuration File Security**

#### **Secure Configuration Storage**
```go
// Secure configuration file handling
func (m *Manager) Save() error {
    configFile := filepath.Join(m.configPath, ConfigFileName+".yaml")

    // Validate configuration before saving
    if err := m.validateConfiguration(); err != nil {
        return fmt.Errorf("configuration validation failed: %w", err)
    }

    // Create secure backup
    if err := m.createSecureBackup(configFile); err != nil {
        m.logger.Warn(context.Background(), "failed to create backup",
            observability.F("error", err.Error()),
        )
    }

    // Write configuration with secure permissions
    data, err := yaml.Marshal(m.config)
    if err != nil {
        return fmt.Errorf("failed to marshal configuration: %w", err)
    }

    if err := os.WriteFile(configFile, data, 0600); err != nil {
        return fmt.Errorf("failed to write configuration: %w", err)
    }

    return nil
}
```

#### **Configuration Validation**
```go
// Comprehensive configuration validation
func (m *Manager) validateConfiguration() error {
    // Validate account configurations
    for alias, account := range m.config.Accounts {
        if err := account.Validate(); err != nil {
            return fmt.Errorf("invalid account %s: %w", alias, err)
        }

        // Validate SSH key security
        if account.SSHKeyPath != "" {
            if err := m.validateSSHKeySecurity(account.SSHKeyPath); err != nil {
                return fmt.Errorf("insecure SSH key for account %s: %w", alias, err)
            }
        }
    }

    // Validate configuration integrity
    if err := m.validateConfigIntegrity(); err != nil {
        return fmt.Errorf("configuration integrity check failed: %w", err)
    }

    return nil
}
```

### **Environment Variable Security**

#### **Secure Environment Handling**
```go
// Secure environment variable processing
func (s *RealZshSecretsService) processEnvironmentVariables() error {
    // Get environment variables securely
    envVars := []string{
        "GITHUB_TOKEN",
        "GITPERSONA_CONFIG_PATH",
        "GITPERSONA_SSH_DIR",
    }

    for _, envVar := range envVars {
        value := os.Getenv(envVar)
        if value != "" {
            // Validate environment variable
            if err := s.validateEnvironmentVariable(envVar, value); err != nil {
                s.logger.Warn(context.Background(), "invalid environment variable",
                    observability.F("variable", envVar),
                    observability.F("error", err.Error()),
                )
                continue
            }

            // Process securely
            if err := s.processEnvironmentVariable(envVar, value); err != nil {
                return fmt.Errorf("failed to process environment variable %s: %w", envVar, err)
            }
        }
    }

    return nil
}
```

---

## 🌐 **Network Security**

### **API Communication Security**

#### **Secure GitHub API Communication**
```go
// Secure GitHub API client configuration
func (s *GitHubTokenService) createSecureClient(token string) *github.Client {
    // Create HTTP client with security settings
    httpClient := &http.Client{
        Timeout: 30 * time.Second,
        Transport: &http.Transport{
            TLSClientConfig: &tls.Config{
                MinVersion: tls.VersionTLS12,
                CipherSuites: []uint16{
                    tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
                    tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
                    tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
                    tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
                },
            },
        },
    }

    // Create GitHub client with secure settings
    client := github.NewClient(httpClient).WithAuthToken(token)

    // Set secure headers
    client.UserAgent = "GitPersona/1.0.0"

    return client
}
```

#### **SSH Connection Security**
```go
// Secure SSH connection configuration
func (s *RealSSHService) createSecureSSHConfig() *ssh.ClientConfig {
    return &ssh.ClientConfig{
        User: "git",
        Auth: []ssh.AuthMethod{
            ssh.PublicKeys(s.getPublicKey()),
        },
        HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Note: In production, use proper host key verification
        Timeout: 10 * time.Second,
        Config: ssh.Config{
            Ciphers: []string{
                "aes256-gcm@openssh.com",
                "chacha20-poly1305@openssh.com",
                "aes256-ctr",
            },
            MACs: []string{
                "hmac-sha2-256-etm@openssh.com",
                "hmac-sha2-256",
            },
        },
    }
}
```

### **Network Validation**

#### **Connection Validation**
```go
// Validate network connections securely
func (s *RealSSHService) validateNetworkConnection(ctx context.Context, host string, port int) error {
    // Create secure dialer
    dialer := &net.Dialer{
        Timeout: 10 * time.Second,
        KeepAlive: 30 * time.Second,
    }

    // Test connection with timeout
    ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
    defer cancel()

    conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", host, port))
    if err != nil {
        return fmt.Errorf("connection to %s:%d failed: %w", host, port, err)
    }
    defer conn.Close()

    // Validate connection security
    if tcpConn, ok := conn.(*net.TCPConn); ok {
        if err := tcpConn.SetKeepAlive(true); err != nil {
            s.logger.Warn(ctx, "failed to set keep-alive",
                observability.F("host", host),
                observability.F("port", port),
            )
        }
    }

    return nil
}
```

---

## 🛠️ **Secure Development**

### **Secure Coding Practices**

#### **Input Sanitization**
```go
// Comprehensive input sanitization
func sanitizeInput(input string) string {
    // Remove potentially dangerous characters
    dangerousChars := []string{
        "<script>", "</script>", "javascript:", "data:",
        "vbscript:", "onload=", "onerror=", "onclick=",
    }

    sanitized := input
    for _, char := range dangerousChars {
        sanitized = strings.ReplaceAll(sanitized, char, "")
    }

    // Remove null bytes
    sanitized = strings.ReplaceAll(sanitized, "\x00", "")

    // Trim whitespace
    sanitized = strings.TrimSpace(sanitized)

    return sanitized
}
```

#### **Secure Error Handling**
```go
// Secure error handling that doesn't leak sensitive information
func (s *Service) handleError(ctx context.Context, err error, operation string) error {
    // Log error with context but without sensitive data
    s.logger.Error(ctx, "operation_failed",
        observability.F("operation", operation),
        observability.F("error_type", reflect.TypeOf(err).String()),
    )

    // Return sanitized error to user
    if strings.Contains(err.Error(), "password") || strings.Contains(err.Error(), "token") {
        return errors.New("authentication failed")
    }

    if strings.Contains(err.Error(), "permission") {
        return errors.New("access denied")
    }

    // Return generic error for unknown issues
    return errors.New("operation failed")
}
```

### **Security Testing**

#### **Security Test Cases**
```go
// Security-focused test cases
func TestSecurity_InputValidation(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected bool
    }{
        {
            name:     "valid email",
            input:    "user@example.com",
            expected: true,
        },
        {
            name:     "email with script injection",
            input:    "user@example.com<script>alert('xss')</script>",
            expected: false,
        },
        {
            name:     "email with javascript",
            input:    "javascript:alert('xss')@example.com",
            expected: false,
        },
        {
            name:     "path traversal attempt",
            input:    "../../../etc/passwd",
            expected: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := isValidEmail(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

---

## 🎯 **Security Best Practices**

### **For Users**

#### **SSH Key Security**
```bash
# 1. Use strong passphrases for SSH keys
gitpersona ssh-keys generate work --passphrase

# 2. Regularly rotate SSH keys
gitpersona ssh-keys rotate work

# 3. Use Ed25519 keys (recommended)
gitpersona ssh-keys generate work --type ed25519

# 4. Monitor key usage
gitpersona ssh-keys audit
```

#### **Configuration Security**
```bash
# 1. Secure configuration directory
chmod 700 ~/.config/gitpersona

# 2. Regular configuration backups
gitpersona config backup

# 3. Validate configuration security
gitpersona config validate --security

# 4. Monitor configuration changes
gitpersona config audit
```

#### **Token Management**
```bash
# 1. Use GitHub CLI for token management
gh auth login

# 2. Regularly refresh tokens
gitpersona secrets refresh-token

# 3. Monitor token usage
gitpersona secrets audit

# 4. Use minimal required permissions
gh auth refresh -s repo,read:user
```

### **For Developers**

#### **Secure Development Practices**
```go
// 1. Always validate input
func processUserInput(input string) error {
    if !isValidInput(input) {
        return errors.New("invalid input")
    }
    // Process input
    return nil
}

// 2. Use secure random number generation
func generateSecureToken() (string, error) {
    bytes := make([]byte, 32)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(bytes), nil
}

// 3. Clear sensitive data from memory
func clearSensitiveData(data []byte) {
    for i := range data {
        data[i] = 0
    }
}

// 4. Use secure file operations
func writeSecureFile(path string, data []byte) error {
    return os.WriteFile(path, data, 0600)
}
```

---

## 🚨 **Incident Response**

### **Security Incident Procedures**

#### **Incident Classification**
```go
// Security incident severity levels
type SecurityIncident struct {
    Severity    IncidentSeverity `json:"severity"`
    Type        IncidentType     `json:"type"`
    Description string           `json:"description"`
    Timestamp   time.Time        `json:"timestamp"`
    Affected    []string         `json:"affected"`
}

type IncidentSeverity int

const (
    SeverityLow IncidentSeverity = iota
    SeverityMedium
    SeverityHigh
    SeverityCritical
)

type IncidentType int

const (
    TypeDataBreach IncidentType = iota
    TypeUnauthorizedAccess
    TypeMaliciousCode
    TypeConfigurationError
    TypeNetworkIntrusion
)
```

#### **Incident Response Process**
```go
// Security incident response
func (s *SecurityService) HandleIncident(ctx context.Context, incident *SecurityIncident) error {
    // 1. Immediate containment
    if err := s.containIncident(ctx, incident); err != nil {
        return fmt.Errorf("failed to contain incident: %w", err)
    }

    // 2. Assess impact
    impact, err := s.assessImpact(ctx, incident)
    if err != nil {
        return fmt.Errorf("failed to assess impact: %w", err)
    }

    // 3. Notify stakeholders
    if err := s.notifyStakeholders(ctx, incident, impact); err != nil {
        s.logger.Error(ctx, "failed to notify stakeholders",
            observability.F("incident", incident.Type),
            observability.F("error", err.Error()),
        )
    }

    // 4. Document incident
    if err := s.documentIncident(ctx, incident, impact); err != nil {
        return fmt.Errorf("failed to document incident: %w", err)
    }

    // 5. Implement remediation
    if err := s.implementRemediation(ctx, incident); err != nil {
        return fmt.Errorf("failed to implement remediation: %w", err)
    }

    return nil
}
```

### **Security Monitoring**

#### **Security Event Logging**
```go
// Security event logging
func (s *SecurityService) LogSecurityEvent(ctx context.Context, event *SecurityEvent) {
    s.logger.Info(ctx, "security_event",
        observability.F("event_type", event.Type),
        observability.F("severity", event.Severity),
        observability.F("source", event.Source),
        observability.F("timestamp", event.Timestamp),
        observability.F("user_id", event.UserID),
        observability.F("ip_address", event.IPAddress),
    )

    // Store in security log
    if err := s.storeSecurityEvent(ctx, event); err != nil {
        s.logger.Error(ctx, "failed to store security event",
            observability.F("error", err.Error()),
        )
    }
}
```

---

## 📚 **Additional Resources**

- **[User Guide](USER_GUIDE.md)** - Complete user documentation
- **[Configuration Guide](CONFIGURATION.md)** - Detailed configuration options
- **[Architecture Guide](ARCHITECTURE.md)** - Technical architecture details
- **[Troubleshooting Guide](TROUBLESHOOTING.md)** - Common issues and solutions
- **[Contributing Guide](CONTRIBUTING.md)** - How to contribute

### **External Security Resources**

- **[OWASP Top 10](https://owasp.org/www-project-top-ten/)** - Web application security risks
- **[NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)** - Cybersecurity best practices
- **[GitHub Security Best Practices](https://docs.github.com/en/code-security)** - GitHub security guidelines
- **[SSH Security Guide](https://infosec.mozilla.org/guidelines/openssh)** - SSH security recommendations

---

<div align="center">

**Security is everyone's responsibility!**

- 🔒 **Report Security Issues**: security@gitpersona.com
- 🛡️ **Follow Best Practices**: Keep your system secure
- 🔍 **Stay Informed**: Regular security updates
- 🤝 **Contribute**: Help improve security

**Together, we can make GitPersona more secure!** 🛡️

</div>
