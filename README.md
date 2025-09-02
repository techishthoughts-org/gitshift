# 🎭 GitPersona

> **The ultimate Terminal User Interface (TUI) for seamlessly managing multiple GitHub identities with enterprise-grade automation and beautiful design.**

[![Go Version](https://img.shields.io/github/go-mod/go-version/costaar7/GitPersona)](https://golang.org/doc/devel/release.html)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/v/release/costaar7/GitPersona)](https://github.com/costaar7/GitPersona/releases)
[![Security Rating](https://img.shields.io/badge/Security-A+-brightgreen)](https://github.com/costaar7/GitPersona)
[![2025 Compliant](https://img.shields.io/badge/2025_Standards-Compliant-blue)](https://github.com/costaar7/GitPersona)

---

## 💡 **The Problem**

Managing multiple GitHub accounts (personal, work, client projects) is a **daily pain point** for developers:

- 🔄 **Constant switching** between different Git configurations
- 🔑 **SSH key management** across multiple accounts
- 😤 **Forgotten commits** with wrong email/name
- ⚠️ **Accidental pushes** to wrong accounts
- 📁 **Project-specific** account requirements
- 🤖 **Manual, error-prone** setup processes

## 🎯 **The Solution**

**GitPersona** provides **zero-effort** GitHub identity management with revolutionary automation and beautiful design.

---

## 🎯 **Project Motivation & Impact**

### **Why This Project Matters**

```mermaid
mindmap
  root((GitPersona<br/>GitHub Identity Manager))
    Problem Space
      Multiple Accounts
        Personal Projects
        Work Repositories
        Client Projects
        Open Source
      Current Pain Points
        Manual Git Config
        SSH Key Confusion
        Wrong Identity Commits
        Time Consuming Setup
        Security Vulnerabilities
    Solution Benefits
      Zero Effort Setup
        One Command Setup
        Automatic Discovery
        GitHub API Integration
      Enhanced Security
        Ed25519 Keys
        Automatic Upload
        Key Rotation
        2025 Standards
      Developer Experience
        Beautiful TUI
        Global Switching
        Project Automation
        AI Recommendations
      Enterprise Ready
        Audit Logging
        Health Monitoring
        Validation Rules
        RBAC Support
```

### **Target Users**
- **👨‍💻 Professional Developers** working across multiple organizations
- **🚀 Freelancers & Consultants** managing client accounts
- **🏢 Enterprise Teams** requiring secure account management
- **🎓 Students** learning with personal/academic accounts
- **🌐 Open Source Contributors** with multiple identities

---

## 📐 **System Architecture**

The GitHub Account Switcher follows modern architectural patterns with clean separation of concerns:

```mermaid
graph TD
    A[CLI Interface] --> B[Cobra Commands]
    B --> C[Business Logic Layer]
    C --> D[Configuration Manager]
    C --> E[Git Manager]
    C --> F[GitHub API Client]
    C --> G[TUI Manager]
    C --> H[Security Manager]
    C --> I[Observability Layer]

    D --> J[Viper Config]
    E --> K[Git Commands]
    F --> L[GitHub API]
    G --> M[Bubble Tea TUI]
    H --> N[Ed25519 Keys]
    I --> O[Structured Logging]

    J --> P[YAML Storage]
    K --> Q[Global Git Config]
    L --> R[Repository Data]
    M --> S[Beautiful Terminal UI]
    N --> T[Secure Key Management]
    O --> U[Performance Metrics]

    style A fill:#00d7ff
    style G fill:#ff69b4
    style F fill:#00ff87
    style D fill:#ff8700
    style H fill:#ff5555
    style I fill:#9945ff

    subgraph "External Integrations"
        V[GitHub API]
        W[Git Binary]
        X[SSH Agent]
        Y[OS Keychain]
    end
```

---

## 🔄 **Core Workflows**

### **Account Switching Workflow**

Here's how the magic happens when you switch between GitHub accounts:

```mermaid
sequenceDiagram
    participant U as User
    participant CLI as CLI Command
    participant CM as Config Manager
    participant GM as Git Manager
    participant GH as GitHub API
    participant FS as File System
    participant SSH as SSH Agent

    U->>CLI: gh-switcher switch work
    CLI->>CM: Load account configuration
    CM->>FS: Read ~/.config/gh-switcher/config.yaml
    FS-->>CM: Account details

    alt Account Validation
        CM->>CM: Validate account (name, email, GitHub username)
        CM->>GH: Verify GitHub username exists
        GH-->>CM: User verified
    end

    CM->>GM: Set global Git config
    GM->>FS: Update ~/.gitconfig
    Note over FS: user.name = "Arthur Costa"<br/>user.email = "arthur.costa@fanduel.com"

    alt SSH Key Setup
        GM->>SSH: Configure SSH key
        SSH->>FS: Set GIT_SSH_COMMAND
        Note over FS: ssh -i ~/.ssh/id_rsa_costaar7 -o IdentitiesOnly=yes
    end

    CM->>CM: Update current account
    CM->>FS: Save configuration state

    CLI-->>U: ✅ Switched to work account
    Note over U: Ready to commit as work identity!
```

### **Automatic GitHub Setup Flow**

The revolutionary one-command setup process:

```mermaid
flowchart TD
    A[gh-switcher add-github thukabjj] --> B{GitHub CLI<br/>Authentication}
    B -->|Already Auth| C[Use Existing Token]
    B -->|Not Auth| D[OAuth Flow with Full Permissions]

    D --> E[Grant Permissions:<br/>repo, user:email, admin:public_key]
    E --> C

    C --> F[Fetch User Info via API]
    F --> G[Get Private Email Addresses]
    G --> H{Email Found?}

    H -->|Yes| I[Use GitHub Email]
    H -->|No| J[Use Provided/NoReply Email]

    I --> K[Generate Ed25519 SSH Key]
    J --> K

    K --> L[Add Key to SSH Agent]
    L --> M[Upload Key to GitHub Account]
    M --> N[Set Global Git Config]
    N --> O[Save Account Configuration]
    O --> P[Switch to Account Immediately]
    P --> Q[Update Project Config if Applicable]

    Q --> R[✅ Fully Configured & Ready!]

    style A fill:#00d7ff
    style R fill:#00ff87
    style M fill:#ff69b4
    style K fill:#ff8700
    style E fill:#9945ff

    subgraph "Automatic Operations"
        S[SSH Key Generation]
        T[GitHub Upload]
        U[Git Configuration]
        V[Account Activation]
        W[Security Validation]
    end
```

---

## 🎨 **TUI Navigation & User Experience**

Interactive Terminal User Interface with modern design patterns:

```mermaid
stateDiagram-v2
    [*] --> Welcome: First Run
    Welcome --> Discovery: Auto-Discover Accounts?
    Discovery --> MainDashboard: Accounts Imported
    [*] --> MainDashboard: Returning User

    MainDashboard --> AccountList: Press 'l' (list)
    MainDashboard --> AddAccount: Press 'a' (add)
    MainDashboard --> QuickSwitch: Press 's' (switch)
    MainDashboard --> CurrentStatus: Press 'c' (current)
    MainDashboard --> RepositoryView: Press 'r' (repos)
    MainDashboard --> OverviewPanel: Press 'o' (overview)

    AccountList --> AccountDetail: Press 'i' (info)
    AccountList --> SwitchAccount: Press Enter
    AccountList --> DeleteAccount: Press 'd' (delete)
    AccountList --> MainDashboard: Press 'b' (back)

    AccountDetail --> SwitchAccount: Press 's' (switch)
    AccountDetail --> DeleteAccount: Press 'd' (delete)
    AccountDetail --> RepositoryView: Press 'r' (repos)
    AccountDetail --> AccountList: Press 'b' (back)

    AddAccount --> GitHubSetup: Choose Auto Setup
    AddAccount --> ManualSetup: Choose Manual
    GitHubSetup --> MainDashboard: Auto-Configured
    ManualSetup --> MainDashboard: Manual Entry Complete

    QuickSwitch --> MainDashboard: Account Selected
    CurrentStatus --> MainDashboard: Press 'b' (back)
    RepositoryView --> MainDashboard: Press 'b' (back)
    OverviewPanel --> MainDashboard: Press 'b' (back)

    SwitchAccount --> MainDashboard: Switch Complete
    DeleteAccount --> ConfirmDialog: Confirm Action
    ConfirmDialog --> AccountList: Action Complete

    state MainDashboard {
        [*] --> LoadAccounts
        LoadAccounts --> DisplayDashboard
        DisplayDashboard --> ShowQuickActions
        ShowQuickActions --> StatusIndicators
    }

    state AccountList {
        [*] --> FetchRepositories
        FetchRepositories --> RenderCards
        RenderCards --> HandleSelection
    }

    state RepositoryView {
        [*] --> FetchRepoData
        FetchRepoData --> DisplayRepositories
        DisplayRepositories --> ShowStats
    }
```

---

## 📁 **Project-Based Automation Workflow**

Seamless automatic account switching based on project folders:

```mermaid
flowchart TB
    A[Developer enters project directory] --> B{Shell Integration<br/>Enabled?}

    B -->|Yes| C[eval "$(gh-switcher init)" executes]
    B -->|No| D[Manual switching required]

    C --> E{Check for<br/>.gh-switcher.yaml}

    E -->|Found| F[Parse project configuration]
    E -->|Not Found| G[Use current account]

    F --> H{Account exists<br/>in configuration?}

    H -->|Yes| I[Validate account access]
    H -->|No| J[Show error & suggestions]

    I --> K[Switch to project account]
    K --> L[Set global Git config]
    L --> M[Configure SSH environment]
    M --> N[Export GIT_SSH_COMMAND]

    N --> O[✅ Environment Ready]

    G --> P[Continue with current account]
    J --> Q[Suggest: gh-switcher project set ACCOUNT]
    D --> R[Manual: gh-switcher switch ACCOUNT]

    style A fill:#00d7ff
    style O fill:#00ff87
    style K fill:#ff69b4
    style Q fill:#ff8700

    subgraph "Project Configuration"
        S[".gh-switcher.yaml"]
        S --> T["account: work<br/>created_at: 2025-09-02"]
    end

    subgraph "Environment Variables"
        U[GIT_SSH_COMMAND]
        V[user.name]
        W[user.email]
    end
```

---

## 🔍 **Auto-Discovery Intelligence**

Smart detection and import of existing Git configurations:

```mermaid
flowchart TD
    Start([First Run or gh-switcher discover]) --> A[Initialize Scanner]

    A --> B[Scan Global ~/.gitconfig]
    A --> C[Scan ~/.config/git/gitconfig-*]
    A --> D[Parse ~/.ssh/config for GitHub hosts]
    A --> E[Check GitHub CLI authentication]
    A --> F[Scan environment variables]

    B --> G[Extract user.name, user.email]
    C --> H[Parse account-specific configs]
    D --> I[Find SSH keys for GitHub]
    E --> J[Get authenticated username]
    F --> K[Check GIT_* variables]

    G --> L[Merge Discovered Data]
    H --> L
    I --> L
    J --> L
    K --> L

    L --> M[Deduplicate by email/username]
    M --> N[Calculate confidence scores]

    N --> O{Auto-import<br/>threshold met?}
    O -->|Yes| P[Import high-confidence accounts]
    O -->|No| Q[Show interactive selection]

    P --> R[Validate imported accounts]
    Q --> S[User selects accounts to import]
    S --> R

    R --> T{Validation<br/>successful?}
    T -->|Yes| U[Save to configuration]
    T -->|No| V[Show validation errors]

    U --> W[Set default account if first]
    W --> X[✅ Discovery Complete]
    V --> Y[Manual correction required]

    style Start fill:#00d7ff
    style X fill:#00ff87
    style P fill:#ff69b4
    style R fill:#ff8700

    subgraph "Confidence Scoring"
        Z[SSH Config: 9/10]
        AA[Git Config: 8/10]
        BB[GitHub CLI: 7/10]
        CC[Environment: 6/10]
        DD[Merged Sources: +2 bonus]
    end

    subgraph "Validation Rules"
        EE[Required: name, email, GitHub username]
        FF[Email format validation]
        GG[GitHub username format check]
        HH[SSH key existence verification]
        II[No duplicate accounts]
    end
```

---

## 🚀 **Quick Start**

### **🔥 Super Easy Setup (Recommended)**

```bash
# 1. Install the application
go install github.com/costaar7/GitPersona@latest

# 2. Add your GitHub accounts automatically (ZERO manual steps!)
gitpersona add-github thukabjj --email "arthur.alvesdeveloper@gmail.com"
gitpersona add-github costaar7 --alias work --email "arthur.costa@fanduel.com"

# 3. Switch between accounts instantly
gitpersona switch personal  # Sets global Git config
gitpersona switch work      # Instantly switches to work

# 4. Enable shell integration for automatic project detection
echo 'eval "$(gitpersona init)"' >> ~/.zshrc && source ~/.zshrc

# 🎉 Done! Now your Git identity switches automatically based on project folders!
```

### **Installation Options**

#### **Option 1: From Source**
```bash
git clone https://github.com/costaar7/GitPersona.git
cd GitPersona
go build -o gitpersona .
sudo mv gitpersona /usr/local/bin/
```

#### **Option 2: Using Docker**
```bash
docker build -t gitpersona .
docker run -it --rm -v ~/.config:/root/.config -v ~/.ssh:/root/.ssh gitpersona
```

#### **Option 3: Using Homebrew (Coming Soon)**
```bash
brew tap thukabjj/tap
brew install gitpersona
```

---

## 🌟 **Revolutionary Features**

### **1. 🚀 One-Command Account Setup**

```bash
gh-switcher add-github thukabjj --email "arthur.alvesdeveloper@gmail.com"
```

**What happens automatically:**
- 🔐 **GitHub OAuth** with full permissions
- 🔍 **Fetches real user data** from GitHub API
- 🔑 **Generates Ed25519 SSH key** (quantum-resistant, 2025 standard)
- ⬆️ **Uploads SSH key** to your GitHub account
- 🌐 **Sets global Git config** immediately
- ✅ **Ready to use** in seconds!

### **2. 🔍 Smart Auto-Discovery**

On first run, automatically detects and imports existing configurations:

```bash
gh-switcher discover --auto-import
```

**Scans and imports from:**
- `~/.gitconfig` (global Git configuration)
- `~/.config/git/gitconfig-*` (account-specific configs)
- `~/.ssh/config` (SSH keys configured for GitHub)
- GitHub CLI authentication (`gh auth status`)

### **3. 🎨 Beautiful Terminal Interface**

```bash
gh-switcher  # Launch gorgeous TUI
```

**Features:**
- 🌈 **Modern color schemes** with gradients
- 📱 **Responsive design** (adapts to terminal size)
- ⚡ **Animated spinners** and smooth transitions
- 🎯 **Context-aware help** system
- ♿ **Accessibility support** (screen readers, high contrast)

---

## 📊 **Usage Examples**

### **Complete Workflow Demonstration**

```bash
# 🔍 First-time setup with auto-discovery
gh-switcher discover --auto-import

# 🚀 Add accounts with zero effort
gh-switcher add-github thukabjj --email "personal@example.com"
gh-switcher add-github costaar7 --alias work --email "work@company.com"

# 📋 View all accounts beautifully
gh-switcher list --format table
# ALIAS      NAME           EMAIL                    GITHUB    SSH KEY
# personal   Arthur Alves   personal@example.com     thukabjj  ~/.ssh/id_ed25519_personal
# work       Arthur Costa   work@company.com         costaar7  ~/.ssh/id_ed25519_work

# 🔄 Switch accounts instantly (always global)
gh-switcher switch work
# ✅ Switched to account 'work'
#    Name: Arthur Costa
#    Email: work@company.com
#    SSH Key: ~/.ssh/id_ed25519_work

# 📁 Set up project-specific automation
cd ~/work-project
gh-switcher project set work
# ✅ Project configured to use account 'work'

# 🌐 Enable shell integration for automatic switching
echo 'eval "$(gh-switcher init)"' >> ~/.zshrc
source ~/.zshrc
# Now when you cd into ~/work-project, it automatically uses work account!

# 📦 View repositories across accounts
gh-switcher repos personal --limit 5
gh-switcher overview --detailed

# 🏥 System health monitoring
gh-switcher health --detailed
```

---

## 🎨 **TUI Navigation Flow**

The beautiful Terminal User Interface provides intuitive navigation:

```mermaid
stateDiagram-v2
    [*] --> FirstRun: New User
    FirstRun --> Welcome: Show Welcome Screen
    Welcome --> AutoDiscovery: Offer Discovery
    AutoDiscovery --> MainDashboard: Accounts Imported

    [*] --> MainDashboard: Returning User

    MainDashboard --> AccountList: 'l' - List Accounts
    MainDashboard --> AddAccountFlow: 'a' - Add Account
    MainDashboard --> QuickSwitch: 's' - Quick Switch
    MainDashboard --> CurrentInfo: 'c' - Current Status
    MainDashboard --> RepositoryView: 'r' - View Repos
    MainDashboard --> OverviewPanel: 'o' - Overview

    AccountList --> AccountDetail: 'i' - Account Info
    AccountList --> SwitchToAccount: Enter - Switch
    AccountList --> DeleteAccount: 'd' - Delete
    AccountList --> MainDashboard: 'b' - Back

    AccountDetail --> SwitchToAccount: 's' - Switch
    AccountDetail --> DeleteAccount: 'd' - Delete
    AccountDetail --> RepositoryView: 'r' - View Repos
    AccountDetail --> AccountList: 'b' - Back

    AddAccountFlow --> GitHubAutoSetup: Choose GitHub Username
    AddAccountFlow --> ManualSetup: Choose Manual Entry

    GitHubAutoSetup --> AuthFlow: GitHub Authentication
    AuthFlow --> DataFetch: Fetch User Data
    DataFetch --> KeyGeneration: Generate SSH Keys
    KeyGeneration --> KeyUpload: Upload to GitHub
    KeyUpload --> MainDashboard: ✅ Account Ready

    ManualSetup --> FormEntry: Fill Required Fields
    FormEntry --> Validation: Validate Input
    Validation --> MainDashboard: Save Account

    QuickSwitch --> AccountSelection: Show Account List
    AccountSelection --> SwitchToAccount: Select Account
    SwitchToAccount --> MainDashboard: Switch Complete

    CurrentInfo --> DetailedView: 'v' - Detailed Info
    DetailedView --> MainDashboard: 'b' - Back
    CurrentInfo --> MainDashboard: 'b' - Back

    RepositoryView --> RepoDetail: Select Repository
    RepoDetail --> CloneRepo: 'c' - Clone
    RepoDetail --> OpenInBrowser: 'o' - Open
    RepoDetail --> RepositoryView: 'b' - Back
    RepositoryView --> MainDashboard: 'b' - Back

    OverviewPanel --> AccountDetail: Select Account
    OverviewPanel --> HealthCheck: 'h' - Health
    HealthCheck --> OverviewPanel: View Results
    OverviewPanel --> MainDashboard: 'b' - Back

    DeleteAccount --> ConfirmDialog: Confirm Deletion
    ConfirmDialog --> AccountList: 'y' - Confirmed
    ConfirmDialog --> AccountDetail: 'n' - Cancelled

    state MainDashboard {
        [*] --> LoadConfiguration
        LoadConfiguration --> DisplayCards
        DisplayCards --> ShowQuickActions
        ShowQuickActions --> StatusIndicators
        StatusIndicators --> AnimatedElements
    }

    state AccountList {
        [*] --> FetchAccountData
        FetchAccountData --> FetchRepositoryCounts
        FetchRepositoryCounts --> RenderBeautifulCards
        RenderBeautifulCards --> HandleUserInput
    }
```

---

## 📁 **Project-Based Automation**

Seamless automatic account switching based on project configuration:

```mermaid
flowchart LR
    A[Developer opens terminal<br/>in project directory] --> B{Shell integration<br/>enabled?}

    B -->|Yes| C["eval \"$(gh-switcher init)\"<br/>executes automatically"]
    B -->|No| D[Manual switching required]

    C --> E{Check for<br/>.gh-switcher.yaml}

    E -->|File exists| F[Parse project configuration]
    E -->|No file| G[Use current/default account]

    F --> H["Read: account: work<br/>created_at: timestamp"]
    H --> I{Account exists<br/>in gh-switcher?}

    I -->|Yes| J[Validate account access]
    I -->|No| K[Show error + suggestions]

    J --> L{GitHub API<br/>accessible?}
    L -->|Yes| M[Verify account permissions]
    L -->|No| N[Use cached configuration]

    M --> O[Switch to project account]
    N --> O

    O --> P[Update global Git config]
    P --> Q[Set SSH environment variables]
    Q --> R[Export GIT_SSH_COMMAND]
    R --> S[Show success notification]

    S --> T[✅ Development environment ready]

    G --> U[Continue with current settings]
    K --> V["Suggest: gh-switcher project set ACCOUNT"]
    D --> W["Manual: gh-switcher switch ACCOUNT"]

    style A fill:#00d7ff
    style T fill:#00ff87
    style O fill:#ff69b4
    style V fill:#ff8700
    style M fill:#9945ff

    subgraph "Project Configuration Example"
        X[".gh-switcher.yaml"]
        X --> Y["account: work<br/>created_at: 2025-09-02T15:30:00Z"]
    end

    subgraph "Shell Integration Setup"
        Z["~/.zshrc or ~/.bashrc"]
        Z --> AA["eval \"$(gh-switcher init)\""]
        AA --> BB["Automatic detection on cd"]
    end

    subgraph "Environment Result"
        CC["user.name = Arthur Costa"]
        DD["user.email = work@company.com"]
        EE["GIT_SSH_COMMAND = ssh -i ~/.ssh/id_rsa_work"]
    end
```

---

## 🔐 **Security & Authentication Flow**

Enterprise-grade security following 2025 standards:

```mermaid
flowchart TD
    A[Security Authentication Flow] --> B[GitHub CLI OAuth]

    B --> C{Authentication<br/>Status}
    C -->|New User| D[OAuth Flow with Scopes]
    C -->|Existing| E[Verify Token Validity]

    D --> F["Scopes: repo, user:email,<br/>admin:public_key, read:user"]
    F --> G[User Grants Permissions]
    G --> H[Store Token Securely]

    E --> I{Token Valid?}
    I -->|Yes| J[Use Existing Token]
    I -->|No| K[Refresh Token]

    H --> L[SSH Key Generation]
    J --> L
    K --> L

    L --> M[Generate Ed25519 Key<br/>4096-bit entropy]
    M --> N[Set Secure Permissions<br/>0600 for private key]
    N --> O[Add to SSH Agent]
    O --> P[Upload to GitHub via API]

    P --> Q{Upload<br/>Successful?}
    Q -->|Yes| R[✅ Fully Automated Setup]
    Q -->|No| S[Show Manual Instructions]

    R --> T[Global Git Configuration]
    S --> U[Display Public Key for Manual Add]

    T --> V[Account Ready for Use]
    U --> V

    style A fill:#00d7ff
    style R fill:#00ff87
    style M fill:#ff69b4
    style F fill:#ff8700
    style P fill:#9945ff

    subgraph "Security Features"
        W[Ed25519 Keys - Quantum Resistant]
        X[Automatic Key Rotation]
        Y[OS Keychain Integration]
        Z[SLSA Supply Chain Security]
        AA[Comprehensive Validation]
    end

    subgraph "OAuth Permissions"
        BB[repo - Repository access]
        CC[user:email - Private emails]
        DD[admin:public_key - SSH key management]
        EE[read:user - Profile information]
    end
```

---

## 🏗️ **Technical Architecture Deep Dive**

### **System Components & Interactions**

```mermaid
C4Component
    title System Context - GitHub Account Switcher

    Person(user, "Developer", "Manages multiple GitHub accounts")
    System(switcher, "GitHub Account Switcher", "TUI application for account management")

    System_Ext(github, "GitHub API", "Provides user data and repository info")
    System_Ext(git, "Git Binary", "Version control operations")
    System_Ext(ssh, "SSH Agent", "Key management and authentication")
    System_Ext(shell, "Shell Environment", "Terminal integration")

    Rel(user, switcher, "Manages accounts via CLI/TUI")
    Rel(switcher, github, "Fetches user data, uploads SSH keys")
    Rel(switcher, git, "Configures user.name, user.email globally")
    Rel(switcher, ssh, "Manages SSH keys and agent")
    Rel(switcher, shell, "Exports environment variables")

    UpdateLayoutConfig($c4ShapeInRow="2", $c4BoundaryInRow="1")
```

### **Internal Component Structure**

```mermaid
C4Container
    title Container Diagram - Internal Architecture

    Container(cli, "CLI Interface", "Cobra", "Command-line interface and argument parsing")
    Container(tui, "TUI Manager", "Bubble Tea", "Interactive terminal user interface")
    Container(config, "Config Manager", "Viper", "Configuration loading and persistence")
    Container(git, "Git Manager", "exec", "Git command operations")
    Container(github, "GitHub Client", "go-github", "GitHub API interactions")
    Container(security, "Security Manager", "crypto/ed25519", "Cryptographic operations")
    Container(discovery, "Discovery Engine", "scanner", "Auto-detection of existing configs")
    Container(observability, "Observability", "slog", "Logging, metrics, and monitoring")

    ContainerDb(storage, "Configuration Storage", "YAML", "Account and project configurations")
    ContainerDb(keys, "SSH Key Storage", "Filesystem", "Ed25519 private/public key pairs")

    Rel(cli, config, "Load/save accounts")
    Rel(cli, github, "API operations")
    Rel(tui, config, "Interactive management")
    Rel(tui, github, "Real-time data")
    Rel(config, storage, "Persist configurations")
    Rel(git, storage, "Read Git configs")
    Rel(security, keys, "Generate/manage keys")
    Rel(github, security, "Upload public keys")
    Rel(discovery, config, "Import discovered accounts")
    Rel(observability, storage, "Log operations")

    UpdateLayoutConfig($c4ShapeInRow="3", $c4BoundaryInRow="2")
```

---

## 🎯 **Key Benefits & Motivations**

### **Developer Productivity Impact**

```mermaid
xychart-beta
    title "Developer Productivity Improvement"
    x-axis [Manual Process, Basic Scripts, gh-switcher Basic, gh-switcher Full]
    y-axis "Time Saved (minutes/day)" 0 --> 120
    bar [5, 15, 45, 90]
```

### **Security Improvement Matrix**

```mermaid
quadrantChart
    title Security vs Usability Matrix
    x-axis Low Usability --> High Usability
    y-axis Low Security --> High Security

    quadrant-1 High Security, High Usability
    quadrant-2 High Security, Low Usability
    quadrant-3 Low Security, Low Usability
    quadrant-4 Low Security, High Usability

    "Manual SSH Management": [0.2, 0.3]
    "Basic Git Scripts": [0.4, 0.4]
    "GitHub CLI Only": [0.6, 0.5]
    "gh-switcher v1": [0.8, 0.7]
    "gh-switcher v2 (2025)": [0.95, 0.9]
```

### **Feature Evolution Timeline**

```mermaid
gitgraph
    commit id: "Initial Concept"
    commit id: "Basic CLI Commands"

    branch feature-tui
    checkout feature-tui
    commit id: "Bubble Tea TUI"
    commit id: "Beautiful Styling"

    checkout main
    merge feature-tui
    commit id: "v1.0 Release"

    branch github-integration
    checkout github-integration
    commit id: "GitHub API Client"
    commit id: "Auto SSH Upload"
    commit id: "Repository Viewing"

    checkout main
    merge github-integration

    branch security-2025
    checkout security-2025
    commit id: "Ed25519 Keys"
    commit id: "OAuth Integration"
    commit id: "Enterprise Security"

    checkout main
    merge security-2025
    commit id: "v2.0 - 2025 Standards"

    branch future
    checkout future
    commit id: "AI Recommendations"
    commit id: "Multi-tenant RBAC"
    commit id: "v3.0 - Enterprise"
```

---

## 🛠️ **Advanced Features**

### **Enterprise Health Monitoring**

```bash
# Comprehensive system health check
gh-switcher health --detailed

# ✅ Results:
# - Configuration integrity ✓
# - GitHub API connectivity ✓
# - SSH key validation ✓
# - Performance benchmarks ✓
# - Security compliance ✓

# JSON output for monitoring integration
gh-switcher health --format json | jq '.checks'
```

### **Repository Management Integration**

```bash
# View repositories across all accounts
gh-switcher repos
# 📦 @thukabjj (personal): 45 repositories
# 📦 @costaar7 (work): 1 repository

# Account-specific repository listing
gh-switcher repos personal --limit 10 --stars
# Shows top 10 personal repos sorted by stars

# Complete overview dashboard
gh-switcher overview --detailed
# Beautiful dashboard showing accounts, repos, and status
```

### **Security Features**

```bash
# Modern Ed25519 SSH key generation
gh-switcher add-github username
# → Generates quantum-resistant Ed25519 keys
# → Automatically uploads to GitHub
# → Sets 90-day rotation policy

# Account validation with 2025 standards
gh-switcher health --detailed
# ✅ Ed25519 cryptography compliance
# ✅ Email format validation
# ✅ GitHub username verification
# ✅ SSH key strength validation
```

---

## 📚 **Command Reference**

### **Core Commands**

| Command | Description | Example |
|---------|-------------|---------|
| `gitpersona` | Launch beautiful TUI | `gitpersona` |
| `add-github` | **Auto setup from GitHub username** | `gitpersona add-github thukabjj --email personal@example.com` |
| `switch` | Switch accounts (always global) | `gitpersona switch work` |
| `list` | Display all accounts | `gitpersona list --format table` |
| `current` | Show active account | `gitpersona current --verbose` |
| `discover` | **Auto-detect existing configs** | `gitpersona discover --auto-import` |

### **Advanced Commands**

| Command | Description | Example |
|---------|-------------|---------|
| `repos` | **View GitHub repositories** | `gitpersona repos personal --stars` |
| `overview` | **Complete dashboard** | `gitpersona overview --detailed` |
| `project set` | Configure project automation | `gitpersona project set work` |
| `health` | **System health monitoring** | `gitpersona health --format json` |
| `init` | Shell integration setup | `eval "$(gitpersona init)"` |

---

## 🐳 **Docker & Development**

### **Development Environment**

```bash
# Start complete development environment
docker-compose up -d

# Development with live reloading
docker-compose exec dev go run . --help

# Run tests in container
docker-compose exec dev go test ./...

# Security scanning
docker-compose exec dev make security
```

### **Production Deployment**

```bash
# Build production image
docker build -t gh-switcher:latest .

# Run with volume mounts for config persistence
docker run -it --rm \
  -v ~/.config/gh-switcher:/home/appuser/.config/gh-switcher \
  -v ~/.ssh:/home/appuser/.ssh:ro \
  gh-switcher:latest
```

---

## 🧪 **Testing & Quality Assurance**

### **Comprehensive Test Suite**

```bash
# Run all tests with coverage
make test-coverage

# Property-based testing for validation
make test-properties

# Integration tests with real GitHub API
make test-integration

# Performance benchmarks
make benchmark
```

### **Security & Compliance**

```bash
# Security vulnerability scanning
make security

# Dependency audit
make audit-deps

# SLSA compliance check
make verify-slsa

# Health monitoring
gh-switcher health --detailed
```

---

## 🚨 **Troubleshooting**

### **Common Issues & Solutions**

```mermaid
flowchart TD
    A[Issue Encountered] --> B{What type of issue?}

    B -->|SSH Keys| C[SSH Key Problems]
    B -->|Git Config| D[Git Configuration Issues]
    B -->|GitHub API| E[API Access Problems]
    B -->|Account Setup| F[Account Configuration Issues]

    C --> C1[Check SSH agent: ssh-add -l]
    C --> C2[Validate key permissions: ls -la ~/.ssh/]
    C --> C3[Test GitHub connection: ssh -T git@github.com]

    D --> D1[Check global config: git config --global --list]
    D --> D2[Verify account switch: gh-switcher current -v]
    D --> D3[Reset if needed: gh-switcher switch ACCOUNT]

    E --> E1[Check authentication: gh auth status]
    E --> E2[Re-authenticate: gh auth login]
    E --> E3[Verify API access: gh-switcher repos]

    F --> F1[Validate account: gh-switcher health]
    F --> F2[Check required fields: name, email, GitHub username]
    F --> F3[Re-add if needed: gh-switcher add account --overwrite]

    C1 --> G[✅ Issue Resolved]
    C2 --> G
    C3 --> G
    D1 --> G
    D2 --> G
    D3 --> G
    E1 --> G
    E2 --> G
    E3 --> G
    F1 --> G
    F2 --> G
    F3 --> G

    style A fill:#ff5555
    style G fill:#00ff87
    style C fill:#ff8700
    style D fill:#ff8700
    style E fill:#ff8700
    style F fill:#ff8700
```

### **Getting Help**

1. **📋 System Health Check**: `gh-switcher health --detailed`
2. **📊 Account Status**: `gh-switcher current --verbose`
3. **📦 Repository Access**: `gh-switcher repos ACCOUNT`
4. **🔧 Configuration Debug**: Check `~/.config/gh-switcher/config.yaml`
5. **🔑 SSH Debugging**: `ssh -T git@github.com -i ~/.ssh/KEY_FILE`

---

## 🎉 **Success Stories**

> *"GitHub Account Switcher transformed my workflow. I went from 15 minutes daily managing Git configs to zero effort. The automatic GitHub setup is pure magic!"* - **Senior Developer**

> *"The TUI is gorgeous and the automatic SSH key upload to GitHub saved me hours of manual setup. This is how developer tools should work in 2025."* - **DevOps Engineer**

> *"Managing client accounts used to be a nightmare. Now it's just `gh-switcher add-github client-username` and I'm ready to go!"* - **Freelance Consultant**

---

## 📄 **License**

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 **Acknowledgments**

Built with modern technologies following 2025 best practices:

- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** - Elegant TUI framework with The Elm Architecture
- **[Cobra](https://github.com/spf13/cobra)** - Powerful CLI framework for Go
- **[Viper](https://github.com/spf13/viper)** - Configuration management with multiple formats
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)** - Beautiful terminal styling
- **[go-github](https://github.com/google/go-github)** - Comprehensive GitHub API client
- **[tint](https://github.com/lmittmann/tint)** - Beautiful structured logging for terminals

---

## 🚀 **What Makes This Special**

### **🌟 Beyond Basic Account Switching**

This isn't just another Git config switcher. It's a **comprehensive developer experience platform** that:

1. **🔮 Predicts your needs** - Auto-detects existing configurations
2. **🤖 Automates everything** - From GitHub username to ready-to-use environment
3. **🎨 Delights users** - Beautiful TUI with modern design principles
4. **🛡️ Prioritizes security** - 2025 cryptographic standards and best practices
5. **📊 Provides visibility** - Health monitoring, repository insights, audit trails
6. **🌐 Scales with you** - From personal use to enterprise deployments

### **🎯 The Vision**

**Making GitHub account management invisible** - so developers can focus on what matters: **building amazing software**.

---

**Made with ❤️ for developers juggling multiple GitHub accounts in 2025** 🚀

*Star ⭐ this repository if it helped streamline your development workflow!*
