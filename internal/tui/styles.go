package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Color palette
var (
	// Primary colors
	PrimaryColor   = lipgloss.Color("#00d7ff")
	SecondaryColor = lipgloss.Color("#ff69b4")
	AccentColor    = lipgloss.Color("#00ff87")

	// Status colors
	SuccessColor = lipgloss.Color("#00ff87")
	WarningColor = lipgloss.Color("#ff8700")
	ErrorColor   = lipgloss.Color("#ff5555")
	InfoColor    = lipgloss.Color("#00d7ff")

	// UI colors
	BackgroundColor = lipgloss.Color("#1a1b26")
	SurfaceColor    = lipgloss.Color("#24283b")
	BorderColor     = lipgloss.Color("#414868")
	TextColor       = lipgloss.Color("#c0caf5")
	MutedColor      = lipgloss.Color("#565f89")

	// Gradient colors
	GradientStart = lipgloss.Color("#7c3aed")
	GradientEnd   = lipgloss.Color("#0ea5e9")
)

// Base styles
var (
	BaseStyle = lipgloss.NewStyle().
			Foreground(TextColor).
			Background(BackgroundColor)

	// Header styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}).
			Padding(0, 1).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			Italic(true)

	// Container styles
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderColor).
			Padding(1, 2).
			MarginBottom(1)

	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderColor).
			Background(SurfaceColor).
			Padding(1, 2).
			MarginBottom(1)

	// Button styles
	ActiveButtonStyle = lipgloss.NewStyle().
				Background(PrimaryColor).
				Foreground(lipgloss.Color("#000000")).
				Bold(true).
				Padding(0, 2).
				Border(lipgloss.RoundedBorder())

	InactiveButtonStyle = lipgloss.NewStyle().
				Background(SurfaceColor).
				Foreground(TextColor).
				Padding(0, 2).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(BorderColor)

	// Status styles
	SuccessStyle = lipgloss.NewStyle().
			Foreground(SuccessColor).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(WarningColor).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(InfoColor).
			Bold(true)

	// List styles
	SelectedItemStyle = lipgloss.NewStyle().
				Background(PrimaryColor).
				Foreground(lipgloss.Color("#000000")).
				Bold(true).
				Padding(0, 1)

	UnselectedItemStyle = lipgloss.NewStyle().
				Foreground(TextColor).
				Padding(0, 1)

	CurrentItemStyle = lipgloss.NewStyle().
				Background(AccentColor).
				Foreground(lipgloss.Color("#000000")).
				Bold(true).
				Padding(0, 1)

	// Form styles
	FocusedInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(PrimaryColor).
				Padding(0, 1)

	BlurredInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(BorderColor).
				Padding(0, 1)

	LabelStyle = lipgloss.NewStyle().
			Foreground(AccentColor).
			Bold(true).
			MarginRight(1)

	// Footer styles
	HelpStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			Italic(true).
			MarginTop(1)

	KeyStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true)

	// Special effects
	GlowStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryColor).
			Background(SurfaceColor).
			Padding(1, 2).
			MarginBottom(1)

	ShadowStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderColor).
			Background(SurfaceColor).
			Padding(1, 2).
			MarginBottom(1).
			MarginLeft(1).
			MarginRight(1)
)

// Utility functions for styling

// RenderTitle creates a beautiful title with gradient effect
func RenderTitle(text string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ffffff")).
		Background(GradientStart).
		Padding(0, 2).
		MarginBottom(1).
		Render("🔄 " + text)
}

// RenderCard creates a styled card container
func RenderCard(title, content string, selected bool) string {
	style := CardStyle
	if selected {
		style = GlowStyle
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(PrimaryColor).
		MarginBottom(1)

	return style.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render(title),
			content,
		),
	)
}

// RenderButton creates a styled button
func RenderButton(text string, active bool) string {
	if active {
		return ActiveButtonStyle.Render(text)
	}
	return InactiveButtonStyle.Render(text)
}

// RenderStatus creates a styled status message
func RenderStatus(text string, statusType string) string {
	var icon, styledText string

	switch statusType {
	case "success":
		icon = "✅"
		styledText = SuccessStyle.Render(text)
	case "warning":
		icon = "⚠️"
		styledText = WarningStyle.Render(text)
	case "error":
		icon = "❌"
		styledText = ErrorStyle.Render(text)
	case "info":
		icon = "ℹ️"
		styledText = InfoStyle.Render(text)
	default:
		icon = "•"
		styledText = text
	}

	return icon + " " + styledText
}

// RenderProgressBar creates a simple progress bar
func RenderProgressBar(current, total int) string {
	if total == 0 {
		return ""
	}

	percentage := float64(current) / float64(total)
	filled := int(percentage * 20) // 20 character width

	bar := ""
	for i := 0; i < 20; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	return lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Render(bar) +
		lipgloss.NewStyle().
			Foreground(MutedColor).
			Render(" "+fmt.Sprintf("%.0f%%", percentage*100))
}

// RenderBorder creates a decorative border
func RenderBorder(content string, width int) string {
	border := strings.Repeat("─", width-2)
	top := "┌" + border + "┐"
	bottom := "└" + border + "┘"

	lines := strings.Split(content, "\n")
	var paddedLines []string

	for _, line := range lines {
		padding := width - lipgloss.Width(line) - 2
		if padding < 0 {
			padding = 0
		}
		paddedLines = append(paddedLines, "│"+line+strings.Repeat(" ", padding)+"│")
	}

	return lipgloss.NewStyle().
		Foreground(BorderColor).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				top,
				strings.Join(paddedLines, "\n"),
				bottom,
			),
		)
}

// Animation states for loading
var (
	SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	DotsFrames    = []string{"   ", ".  ", ".. ", "..."}
)

// RenderSpinner creates an animated spinner
func RenderSpinner(frame int) string {
	return lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Render(SpinnerFrames[frame%len(SpinnerFrames)])
}

// RenderDashboard creates the main dashboard layout
func RenderDashboard(currentAccount string, accountCount int, width int) string {
	// Create status cards
	var statusCards []string

	// Current account card
	currentCard := RenderCard(
		"Current Account",
		func() string {
			if currentAccount != "" {
				return SuccessStyle.Render("✓ " + currentAccount)
			}
			return WarningStyle.Render("✗ No account active")
		}(),
		false,
	)

	// Account count card
	countCard := RenderCard(
		"Total Accounts",
		InfoStyle.Render(fmt.Sprintf("📊 %d configured", accountCount)),
		false,
	)

	statusCards = append(statusCards, currentCard, countCard)

	// Join cards horizontally if there's enough width
	if width > 80 {
		return lipgloss.JoinHorizontal(lipgloss.Top, statusCards...)
	} else {
		return lipgloss.JoinVertical(lipgloss.Left, statusCards...)
	}
}

// RenderMenu creates a beautiful menu with options
func RenderMenu(options []string, selected int) string {
	var renderedOptions []string

	for i, option := range options {
		if i == selected {
			renderedOptions = append(renderedOptions, SelectedItemStyle.Render("▶ "+option))
		} else {
			renderedOptions = append(renderedOptions, UnselectedItemStyle.Render("  "+option))
		}
	}

	return PanelStyle.Render(lipgloss.JoinVertical(lipgloss.Left, renderedOptions...))
}

// RenderHelpBar creates a help bar with key bindings
func RenderHelpBar(bindings map[string]string) string {
	var helpItems []string

	for key, desc := range bindings {
		helpItems = append(helpItems, KeyStyle.Render(key)+" "+desc)
	}

	return HelpStyle.Render(strings.Join(helpItems, "  •  "))
}
