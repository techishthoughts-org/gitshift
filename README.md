# 🎭 gitshift

> **SSH-First GitHub Account Management - Clean, Fast, and Isolated**

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue)](https://golang.org/doc/devel/release.html)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen)](#)

---

## 🎯 **What is gitshift?**

gitshift is a **clean, focused CLI tool** for managing multiple GitHub accounts with **complete SSH isolation**. No GitHub API dependencies, no complex TUI interfaces - just pure SSH-based account management that works.

### **The Problem We Solve**
- 🔄 **Manual switching** between Git configurations
- 🔑 **SSH key conflicts** across multiple accounts
- 😤 **Wrong commits** pushed to wrong accounts
- 🚫 **Key interference** and authentication failures
- 📁 **Complex SSH setup** for each account
- 🤔 **Hard-to-manage** known_hosts entries

### **Our SSH-First Solution**
**gitshift** provides **complete isolation** with:
- 🔐 **SSH-Only Approach** - No GitHub API dependencies
- 🔄 **Complete Isolation** - Accounts never interfere
- 🔑 **Smart SSH Management** - Auto-generates and manages keys
- ⚡ **Fast Switching** - Instant account transitions
- 🛡️ **Secure by Design** - SSH config with `IdentitiesOnly=yes`
- 🌐 **Known Hosts Management** - Auto-manages GitHub host keys
---

## 🚀 **Quick Start**

### **Installation**

```bash
# Clone and build
git clone https://github.com/techishthoughts/gitshift.git
cd gitshift
make build
```

Or using Go:
```bash
go install github.com/techishthoughts/gitshift@latest
```

### **Basic Usage**

1. **Discover existing SSH keys:**
```bash
gitshift discover
```

2. **Generate a new SSH key:**
```bash
gitshift ssh-keygen work --email work@company.com
gitshift ssh-keygen personal --email me@gmail.com --type ed25519
```

3. **List accounts:**
```bash
gitshift list
```

4. **Switch accounts:**
```bash
gitshift switch work
gitshift switch personal
```

5. **Test SSH connectivity:**
```bash
gitshift ssh-test
gitshift ssh-test work --verbose
```

---

## 🔧 **Core Commands**

### **Account Management**
- `gitshift add` - Add a new account manually
- `gitshift list` - List all configured accounts
- `gitshift remove` - Remove an account
- `gitshift update` - Update account information
- `gitshift switch` - Switch to a different account

### **SSH Key Management**
- `gitshift ssh-keygen` - Generate SSH keys with full control
- `gitshift ssh-test` - Test and troubleshoot SSH connectivity
- `gitshift discover` - Auto-discover accounts from SSH keys

### **SSH Key Generation Options**
```bash
# Ed25519 key (recommended)
gitshift ssh-keygen myaccount --email me@example.com

# RSA key with custom size
gitshift ssh-keygen myaccount --type rsa --bits 4096

# With passphrase protection
gitshift ssh-keygen myaccount --passphrase "my-secure-password"

# Auto-add to GitHub (requires gh CLI)
gitshift ssh-keygen myaccount --add-to-github

# Force overwrite existing key
gitshift ssh-keygen myaccount --force
```

---

## 🏗️ **Architecture**

gitshift uses a **pure SSH-first approach**:

### **SSH Isolation Strategy**
- **Unique SSH keys** per account (`id_ed25519_accountname`)
- **SSH config** with `IdentitiesOnly=yes` for strict isolation
- **SSH agent management** - loads only required keys
- **Known hosts** auto-management for GitHub

### **No External Dependencies**
- ✅ **No GitHub API calls** - works purely with SSH
- ✅ **No complex TUI** - clean CLI interface
- ✅ **No caching issues** - direct SSH directory scanning
- ✅ **No auto-generation** - only manages existing keys

### **File Structure**
```
~/.ssh/
├── id_ed25519_work          # Work account private key
├── id_ed25519_work.pub      # Work account public key
├── id_ed25519_personal      # Personal account private key
├── id_ed25519_personal.pub  # Personal account public key
├── config                   # SSH config with isolation
└── known_hosts              # GitHub host keys

~/.config/gitshift/
└── config.yaml              # gitshift configuration
```

---

## 🔒 **Security Features**

### **SSH Security**
- **Ed25519 keys** by default (most secure)
- **Proper permissions** (600 for private, 644 for public)
- **IdentitiesOnly=yes** prevents key leakage
- **Current GitHub host keys** (2024)

### **Account Isolation**
- **Complete SSH isolation** between accounts
- **No key conflicts** or cross-contamination
- **SSH agent clearing** before key loading
- **Config backup** before modifications

---

## 🛠️ **Development**

### **Build from Source**
```bash
git clone https://github.com/techishthoughts/gitshift.git
cd gitshift
make dev    # Full development build with tests
```

### **Available Make Targets**
- `make build` - Build the binary
- `make test` - Run tests
- `make dev` - Full development workflow
- `make demo` - Show gitshift in action
- `make clean` - Clean build artifacts
- `make release` - Cross-platform release builds

### **Project Structure**
```
gitshift/
├── cmd/                    # CLI commands
│   ├── add.go             # Account addition
│   ├── discover.go        # SSH key discovery
│   ├── list.go            # Account listing
│   ├── ssh-keygen.go      # SSH key generation
│   ├── ssh-test.go        # SSH testing
│   └── switch.go          # Account switching
├── internal/
│   ├── config/            # Configuration management
│   ├── discovery/         # SSH-only discovery
│   ├── git/               # Git operations
│   ├── models/            # Data models
│   └── ssh/               # SSH management
└── main.go                # Entry point
```

---

## 🤝 **Contributing**

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make changes and test: `make dev`
4. Commit: `git commit -m 'Add amazing feature'`
5. Push: `git push origin feature/amazing-feature`
6. Open a Pull Request

---

## 📄 **License**

MIT License - see [LICENSE](LICENSE) file for details.

---

## 🙏 **Acknowledgments**

- Built with ❤️ for developers managing multiple GitHub accounts
- Inspired by the need for **simple, secure, SSH-first** account management
- Thanks to the Go community for excellent CLI tools and libraries

---

**gitshift: Where SSH simplicity meets GitHub productivity** 🎭✨
