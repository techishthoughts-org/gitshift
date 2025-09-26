package internal

import (
	"fmt"
	"strings"
	"time"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// TUI provides terminal user interface components with progressive disclosure
type TUI struct {
	logger observability.Logger
	width  int
	theme  *Theme
}

// Theme defines the visual appearance of TUI components
type Theme struct {
	Primary     string // Primary color (usually blue)
	Secondary   string // Secondary color (usually green)
	Warning     string // Warning color (usually yellow)
	Error       string // Error color (usually red)
	Info        string // Info color (usually cyan)
	Muted       string // Muted color (usually gray)
	Background  string // Background color
	Foreground  string // Foreground color
	BorderStyle string // Border style for boxes
}

// ProgressiveDisplay handles progressive disclosure of information
type ProgressiveDisplay struct {
	tui      *TUI
	level    DisplayLevel
	maxItems int
	showMore bool
	items    []DisplayItem
}

// DisplayLevel represents different levels of detail
type DisplayLevel string

const (
	DisplayLevelBasic    DisplayLevel = "basic"
	DisplayLevelDetailed DisplayLevel = "detailed"
	DisplayLevelVerbose  DisplayLevel = "verbose"
)

// DisplayItem represents an item in a progressive display
type DisplayItem struct {
	Title      string
	Content    string
	Level      DisplayLevel
	Icon       string
	Status     string
	Actions    []string
	Metadata   map[string]interface{}
	Expandable bool
	Expanded   bool
}

// StatusIndicator provides various status icons and formatting
type StatusIndicator struct {
	Status  string
	Icon    string
	Color   string
	Message string
}

// DefaultTheme provides a default color theme
func DefaultTheme() *Theme {
	return &Theme{
		Primary:     "\033[34m", // Blue
		Secondary:   "\033[32m", // Green
		Warning:     "\033[33m", // Yellow
		Error:       "\033[31m", // Red
		Info:        "\033[36m", // Cyan
		Muted:       "\033[90m", // Gray
		Background:  "\033[40m", // Black background
		Foreground:  "\033[37m", // White foreground
		BorderStyle: "─│┌┐└┘",
	}
}

// NewTUI creates a new TUI instance
func NewTUI(logger observability.Logger) *TUI {
	return &TUI{
		logger: logger,
		width:  80, // Default width
		theme:  DefaultTheme(),
	}
}

// SetWidth sets the display width
func (t *TUI) SetWidth(width int) {
	t.width = width
}

// SetTheme sets the color theme
func (t *TUI) SetTheme(theme *Theme) {
	t.theme = theme
}

// ShowHeader displays a formatted header
func (t *TUI) ShowHeader(title, subtitle string) {
	t.printSeparator("=")
	t.printCentered(title, t.theme.Primary)
	if subtitle != "" {
		t.printCentered(subtitle, t.theme.Muted)
	}
	t.printSeparator("=")
	fmt.Println()
}

// ShowSection displays a section header
func (t *TUI) ShowSection(title string, level DisplayLevel) {
	icon := t.getSectionIcon(level)
	fmt.Printf("%s%s %s%s\n", t.theme.Secondary, icon, title, t.reset())
	t.printSeparator("-")
}

// ShowStatus displays a status with icon and color
func (t *TUI) ShowStatus(status string, message string, details []string) {
	indicator := t.getStatusIndicator(status)
	fmt.Printf("%s%s %s%s\n", indicator.Color, indicator.Icon, message, t.reset())

	if len(details) > 0 && status != "success" {
		for _, detail := range details {
			fmt.Printf("   %s• %s%s\n", t.theme.Muted, detail, t.reset())
		}
	}
}

// ShowAccountList displays a list of accounts with progressive disclosure
func (t *TUI) ShowAccountList(accounts []*Account, currentAccount string, level DisplayLevel) {
	display := t.NewProgressiveDisplay(level)

	for _, account := range accounts {
		item := DisplayItem{
			Title: account.Alias,
			Level: DisplayLevelBasic,
			Icon:  "👤",
		}

		// Current account indicator
		if account.Alias == currentAccount {
			item.Icon = "▶️"
			item.Status = "current"
		}

		// Basic information
		item.Content = fmt.Sprintf("%s <%s>", account.Name, account.Email)

		// Detailed information
		if level >= DisplayLevelDetailed {
			details := []string{}
			if account.GitHubUsername != "" {
				details = append(details, fmt.Sprintf("GitHub: @%s", account.GitHubUsername))
			}
			if account.SSHKeyPath != "" {
				details = append(details, fmt.Sprintf("SSH: %s", account.SSHKeyPath))
			}
			if account.LastUsed != nil {
				details = append(details, fmt.Sprintf("Last used: %s", *account.LastUsed))
			}

			if len(details) > 0 {
				item.Content += "\n   " + strings.Join(details, "\n   ")
			}
		}

		display.AddItem(item)
	}

	display.Render()
}

// ShowDiagnosticResults displays diagnostic results with progressive disclosure
func (t *TUI) ShowDiagnosticResults(results *DiagnosticsReport, level DisplayLevel) {
	t.ShowHeader("Diagnostic Results", fmt.Sprintf("Overall Status: %s", results.Overall))

	display := t.NewProgressiveDisplay(level)

	for _, check := range results.Checks {
		item := DisplayItem{
			Title: check.Name,
			Level: DisplayLevelBasic,
			Icon:  t.getStatusIcon(check.Status),
		}

		item.Content = check.Message

		if level >= DisplayLevelDetailed && check.Fix != "" {
			item.Content += fmt.Sprintf("\n   💡 Fix: %s", check.Fix)
		}

		if check.Status != "pass" {
			item.Status = check.Status
		}

		display.AddItem(item)
	}

	display.Render()

	// Show summary
	t.ShowSection("Summary", DisplayLevelBasic)
	fmt.Printf("Total checks: %d\n", len(results.Checks))
	if results.Overall != "healthy" {
		fmt.Printf("%s%s Overall status needs attention%s\n",
			t.theme.Warning, "⚠️", t.reset())
	}
}

// ShowSSHKeyList displays SSH keys with status
func (t *TUI) ShowSSHKeyList(keys []*SSHKeyInfo, level DisplayLevel) {
	t.ShowHeader("SSH Keys", fmt.Sprintf("%d keys found", len(keys)))

	display := t.NewProgressiveDisplay(level)

	for _, key := range keys {
		item := DisplayItem{
			Title: key.Path,
			Level: DisplayLevelBasic,
			Icon:  "🔑",
		}

		if key.Valid {
			item.Icon = "✅"
			item.Status = "valid"
		} else {
			item.Icon = "❌"
			item.Status = "invalid"
		}

		// Basic info
		item.Content = fmt.Sprintf("%s (%d bits)", key.Type, key.Size)

		// Detailed info
		if level >= DisplayLevelDetailed {
			details := []string{}
			details = append(details, fmt.Sprintf("Fingerprint: %s", key.Fingerprint))
			if key.Email != "" {
				details = append(details, fmt.Sprintf("Email: %s", key.Email))
			}
			item.Content += "\n   " + strings.Join(details, "\n   ")
		}

		display.AddItem(item)
	}

	display.Render()
}

// ShowInteractivePrompt displays an interactive prompt with options
func (t *TUI) ShowInteractivePrompt(title string, options []string, defaultOption int) int {
	t.ShowSection(title, DisplayLevelBasic)

	for i, option := range options {
		indicator := " "
		if i == defaultOption {
			indicator = "▶"
		}
		fmt.Printf("  %s %d. %s\n", indicator, i+1, option)
	}

	fmt.Print("\nSelect option: ")
	// This would integrate with actual input handling
	return defaultOption
}

// ShowProgress displays a progress indicator
func (t *TUI) ShowProgress(message string, current, total int) {
	if total == 0 {
		fmt.Printf("\r%s %s... ", t.theme.Info+"⏳"+t.reset(), message)
		return
	}

	percentage := float64(current) / float64(total) * 100
	barWidth := 30
	filled := int(percentage / 100 * float64(barWidth))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	fmt.Printf("\r%s%s %s [%s] %d/%d (%.1f%%)%s",
		t.theme.Info, "⏳", message, bar, current, total, percentage, t.reset())

	if current == total {
		fmt.Println() // New line when complete
	}
}

// NewProgressiveDisplay creates a new progressive display
func (t *TUI) NewProgressiveDisplay(level DisplayLevel) *ProgressiveDisplay {
	return &ProgressiveDisplay{
		tui:      t,
		level:    level,
		maxItems: t.getMaxItemsForLevel(level),
		items:    []DisplayItem{},
	}
}

// AddItem adds an item to the progressive display
func (pd *ProgressiveDisplay) AddItem(item DisplayItem) {
	pd.items = append(pd.items, item)
}

// Render displays the items according to the current level
func (pd *ProgressiveDisplay) Render() {
	visibleItems := pd.getVisibleItems()

	for i, item := range visibleItems {
		pd.renderItem(item, i)
	}

	// Show "more" indicator if needed
	if len(pd.items) > len(visibleItems) {
		hidden := len(pd.items) - len(visibleItems)
		fmt.Printf("\n%s... and %d more (use --detailed or --verbose for more info)%s\n",
			pd.tui.theme.Muted, hidden, pd.tui.reset())
	}
}

// Private methods

func (pd *ProgressiveDisplay) getVisibleItems() []DisplayItem {
	var visible []DisplayItem

	for _, item := range pd.items {
		// Include item if it matches the current display level
		if item.Level <= pd.level || pd.level == DisplayLevelVerbose {
			visible = append(visible, item)
		}

		// Respect max items limit for basic level
		if pd.level == DisplayLevelBasic && len(visible) >= pd.maxItems {
			break
		}
	}

	return visible
}

func (pd *ProgressiveDisplay) renderItem(item DisplayItem, index int) {
	// Item header
	fmt.Printf("%s %s", item.Icon, item.Title)

	// Status indicator
	if item.Status != "" {
		indicator := pd.tui.getStatusIndicator(item.Status)
		fmt.Printf(" %s%s%s", indicator.Color, indicator.Icon, pd.tui.reset())
	}

	fmt.Println()

	// Item content
	if item.Content != "" {
		contentLines := strings.Split(item.Content, "\n")
		for _, line := range contentLines {
			if strings.TrimSpace(line) != "" {
				fmt.Printf("   %s\n", line)
			}
		}
	}

	// Actions (for interactive mode)
	if len(item.Actions) > 0 && pd.level >= DisplayLevelDetailed {
		fmt.Printf("   %sActions: %s%s\n", pd.tui.theme.Muted, strings.Join(item.Actions, ", "), pd.tui.reset())
	}

	fmt.Println()
}

func (t *TUI) printSeparator(char string) {
	fmt.Println(strings.Repeat(char, t.width))
}

func (t *TUI) printCentered(text, color string) {
	padding := (t.width - len(text)) / 2
	if padding > 0 {
		fmt.Printf("%s%s%s%s\n", strings.Repeat(" ", padding), color, text, t.reset())
	} else {
		fmt.Printf("%s%s%s\n", color, text, t.reset())
	}
}

func (t *TUI) getSectionIcon(level DisplayLevel) string {
	switch level {
	case DisplayLevelBasic:
		return "📋"
	case DisplayLevelDetailed:
		return "📊"
	case DisplayLevelVerbose:
		return "🔬"
	default:
		return "•"
	}
}

func (t *TUI) getStatusIcon(status string) string {
	switch status {
	case "pass", "success", "valid":
		return "✅"
	case "fail", "error", "invalid":
		return "❌"
	case "warn", "warning":
		return "⚠️"
	case "info":
		return "ℹ️"
	case "pending":
		return "⏳"
	case "current", "active":
		return "▶️"
	default:
		return "•"
	}
}

func (t *TUI) getStatusIndicator(status string) StatusIndicator {
	indicators := map[string]StatusIndicator{
		"success": {Icon: "✅", Color: t.theme.Secondary, Message: "Success"},
		"error":   {Icon: "❌", Color: t.theme.Error, Message: "Error"},
		"warning": {Icon: "⚠️", Color: t.theme.Warning, Message: "Warning"},
		"info":    {Icon: "ℹ️", Color: t.theme.Info, Message: "Info"},
		"pending": {Icon: "⏳", Color: t.theme.Primary, Message: "Pending"},
		"current": {Icon: "▶️", Color: t.theme.Primary, Message: "Current"},
	}

	if indicator, exists := indicators[status]; exists {
		return indicator
	}

	return StatusIndicator{Icon: "•", Color: t.theme.Foreground, Message: status}
}

func (t *TUI) getMaxItemsForLevel(level DisplayLevel) int {
	switch level {
	case DisplayLevelBasic:
		return 5
	case DisplayLevelDetailed:
		return 20
	case DisplayLevelVerbose:
		return 1000 // Effectively unlimited
	default:
		return 10
	}
}

func (t *TUI) reset() string {
	return "\033[0m"
}

// Utility functions for common TUI patterns

// ShowErrorWithSuggestions displays an error with helpful suggestions
func (t *TUI) ShowErrorWithSuggestions(err *GitPersonaError) {
	indicator := t.getStatusIndicator("error")
	fmt.Printf("%s%s %s%s\n", indicator.Color, indicator.Icon, err.UserMessage, t.reset())

	if len(err.Suggestions) > 0 {
		fmt.Printf("\n💡 Suggestions:\n")
		for i, suggestion := range err.Suggestions {
			fmt.Printf("   %d. %s\n", i+1, suggestion)
		}
	}

	if err.Code != "" {
		fmt.Printf("\n%s🔍 Error Code: %s%s\n", t.theme.Muted, err.Code, t.reset())
	}
}

// ShowSuccessMessage displays a success message
func (t *TUI) ShowSuccessMessage(message string, details []string) {
	t.ShowStatus("success", message, details)
}

// ShowLoadingSpinner displays a loading spinner (mock implementation)
func (t *TUI) ShowLoadingSpinner(message string, duration time.Duration) {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	start := time.Now()

	for time.Since(start) < duration {
		for _, frame := range frames {
			fmt.Printf("\r%s%s %s%s", t.theme.Info, frame, message, t.reset())
			time.Sleep(100 * time.Millisecond)
			if time.Since(start) >= duration {
				break
			}
		}
	}

	fmt.Printf("\r%s✅ %s completed%s\n", t.theme.Secondary, message, t.reset())
}
