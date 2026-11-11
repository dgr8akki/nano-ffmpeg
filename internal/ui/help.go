package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// HelpEntry describes a single help item.
type HelpEntry struct {
	Key  string
	Desc string
}

// HelpSection is a titled group of help entries.
type HelpSection struct {
	Title   string
	Entries []HelpEntry
}

// HelpOverlay renders a centered help overlay.
func HelpOverlay(sections []HelpSection, width, height int) string {
	var content strings.Builder

	title := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Render("Help")
	content.WriteString(title + "\n\n")

	for _, sec := range sections {
		sectionTitle := lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true).
			Render(sec.Title)
		content.WriteString(sectionTitle + "\n")

		for _, e := range sec.Entries {
			key := lipgloss.NewStyle().
				Foreground(ColorText).
				Bold(true).
				Width(12).
				Render(e.Key)
			desc := lipgloss.NewStyle().
				Foreground(ColorDim).
				Render(e.Desc)
			content.WriteString("  " + key + desc + "\n")
		}
		content.WriteString("\n")
	}

	content.WriteString(lipgloss.NewStyle().
		Foreground(ColorMuted).
		Render("Press ? or Esc to close"))

	// Box it
	boxWidth := 50
	if width > 0 && boxWidth > width-4 {
		boxWidth = width - 4
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(1, 2).
		Width(boxWidth).
		Render(content.String())

	// Center vertically and horizontally
	boxHeight := lipgloss.Height(box)
	padTop := (height - boxHeight) / 2
	if padTop < 0 {
		padTop = 0
	}
	padLeft := (width - lipgloss.Width(box)) / 2
	if padLeft < 0 {
		padLeft = 0
	}

	return strings.Repeat("\n", padTop) +
		lipgloss.NewStyle().PaddingLeft(padLeft).Render(box)
}

// HomeHelp returns help for the home screen.
func HomeHelp() []HelpSection {
	return []HelpSection{
		{
			Title: "Navigation",
			Entries: []HelpEntry{
				{Key: "↑ / k", Desc: "Move up"},
				{Key: "↓ / j", Desc: "Move down"},
				{Key: "Enter", Desc: "Select operation & pick file"},
				{Key: "q", Desc: "Quit"},
				{Key: "?", Desc: "Toggle this help"},
			},
		},
	}
}

// FilePickerHelp returns help for the file picker.
func FilePickerHelp() []HelpSection {
	return []HelpSection{
		{
			Title: "File Browser",
			Entries: []HelpEntry{
				{Key: "↑ / k", Desc: "Move up"},
				{Key: "↓ / j", Desc: "Move down"},
				{Key: "Enter", Desc: "Open directory / select file"},
				{Key: "Backspace", Desc: "Go to parent directory"},
				{Key: "/", Desc: "Switch to path input mode"},
				{Key: "Esc", Desc: "Go back"},
			},
		},
		{
			Title: "Path Input Mode",
			Entries: []HelpEntry{
				{Key: "Enter", Desc: "Navigate to path"},
				{Key: "Esc", Desc: "Cancel path input"},
			},
		},
	}
}

// OperationsHelp returns help for the operations screen.
func OperationsHelp() []HelpSection {
	return []HelpSection{
		{
			Title: "Operations",
			Entries: []HelpEntry{
				{Key: "↑ / k", Desc: "Move up"},
				{Key: "↓ / j", Desc: "Move down"},
				{Key: "Enter", Desc: "Select operation"},
				{Key: "Esc", Desc: "Go back to file picker"},
			},
		},
	}
}

// SettingsHelp returns help for the settings screen.
func SettingsHelp() []HelpSection {
	return []HelpSection{
		{
			Title: "Settings",
			Entries: []HelpEntry{
				{Key: "↑ / k", Desc: "Previous field"},
				{Key: "↓ / j", Desc: "Next field"},
				{Key: "← / →", Desc: "Change value / toggle"},
				{Key: "Enter", Desc: "Execute ffmpeg command"},
				{Key: "Esc", Desc: "Go back"},
			},
		},
	}
}

// ProgressHelp returns help for the progress screen.
func ProgressHelp() []HelpSection {
	return []HelpSection{
		{
			Title: "Encoding",
			Entries: []HelpEntry{
				{Key: "Esc", Desc: "Cancel encoding (with confirmation)"},
			},
		},
	}
}
