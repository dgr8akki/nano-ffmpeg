package result

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/dgr8akki/nano-ffmpeg/internal/ffmpeg"
	"github.com/dgr8akki/nano-ffmpeg/internal/screens"
	"github.com/dgr8akki/nano-ffmpeg/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// buyMeACoffeeURL is the support link opened from the result screen.
const buyMeACoffeeURL = "https://buymeacoffee.com/dgr8akki"

// openURL opens the given URL in the user's default browser. It is declared as
// a variable so tests can stub it out.
var openURL = func(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// Model is the result screen model.
type Model struct {
	outputPath string
	inputSize  int64
	outputSize int64
	cursor     int
	options    []string
	width      int
	height     int
}

// New creates a new result screen.
func New(outputPath string, inputSize int64) *Model {
	var outputSize int64
	if info, err := os.Stat(outputPath); err == nil {
		outputSize = info.Size()
	}

	return &Model{
		outputPath: outputPath,
		inputSize:  inputSize,
		outputSize: outputSize,
		options:    []string{"Do another operation", "Buy me a coffee ☕", "Quit"},
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case "enter":
			switch m.cursor {
			case 0:
				return m, func() tea.Msg {
					return screens.NavigateMsg{Screen: screens.ScreenHome}
				}
			case 1:
				return m, func() tea.Msg {
					_ = openURL(buyMeACoffeeURL)
					return nil
				}
			case 2:
				return m, tea.Quit
			}
		case "esc":
			return m, func() tea.Msg {
				return screens.NavigateMsg{Screen: screens.ScreenHome}
			}
		}
	}
	return m, nil
}

func (m *Model) View() string {
	var b strings.Builder

	// Success header
	check := lipgloss.NewStyle().
		Foreground(ui.ColorSuccess).
		Bold(true).
		Render("  Done!")
	b.WriteString(check)
	b.WriteString("\n\n")

	// Output file path
	pathLabel := lipgloss.NewStyle().Foreground(ui.ColorDim).Render("  Output: ")
	pathValue := lipgloss.NewStyle().Foreground(ui.ColorSecondary).Bold(true).Render(m.outputPath)
	b.WriteString(pathLabel + pathValue)
	b.WriteString("\n\n")

	// Size comparison
	b.WriteString(m.renderSizeComparison())
	b.WriteString("\n\n")

	// Options
	for i, opt := range m.options {
		if i == m.cursor {
			indicator := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).Render(" > ")
			name := ui.SelectedStyle.Render(opt)
			b.WriteString(indicator + name + "\n")
		} else {
			name := lipgloss.NewStyle().Foreground(ui.ColorText).PaddingLeft(3).Render(opt)
			b.WriteString(name + "\n")
		}
	}

	return b.String()
}

func (m *Model) renderSizeComparison() string {
	if m.inputSize == 0 || m.outputSize == 0 {
		return ""
	}

	inputStr := ffmpeg.FormatSize(m.inputSize)
	outputStr := ffmpeg.FormatSize(m.outputSize)

	ratio := float64(m.outputSize) / float64(m.inputSize)
	var changeStr string
	var changeStyle lipgloss.Style

	if ratio < 1 {
		pct := (1 - ratio) * 100
		changeStr = fmt.Sprintf("%.1f%% smaller", pct)
		changeStyle = lipgloss.NewStyle().Foreground(ui.ColorSuccess).Bold(true)
	} else {
		pct := (ratio - 1) * 100
		changeStr = fmt.Sprintf("%.1f%% larger", pct)
		changeStyle = lipgloss.NewStyle().Foreground(ui.ColorWarning).Bold(true)
	}

	// Visual bar
	barWidth := 30
	inputBar := lipgloss.NewStyle().Foreground(ui.ColorMuted).Render(strings.Repeat("█", barWidth))
	outputBarLen := int(float64(barWidth) * ratio)
	if outputBarLen > barWidth {
		outputBarLen = barWidth
	}
	if outputBarLen < 1 {
		outputBarLen = 1
	}
	outputBarColor := ui.ColorSuccess
	if ratio >= 1 {
		outputBarColor = ui.ColorWarning
	}
	outputBar := lipgloss.NewStyle().Foreground(outputBarColor).Render(strings.Repeat("█", outputBarLen))
	outputPad := strings.Repeat(" ", barWidth-outputBarLen)

	col := lipgloss.NewStyle().Foreground(ui.ColorDim).PaddingLeft(2)

	lines := []string{
		col.Render(fmt.Sprintf("Input:  %s  %s", inputBar, inputStr)),
		col.Render(fmt.Sprintf("Output: %s%s  %s", outputBar, outputPad, outputStr)),
		"",
		lipgloss.NewStyle().PaddingLeft(2).Render(changeStyle.Render("  " + changeStr)),
	}

	return strings.Join(lines, "\n")
}

func (m *Model) Breadcrumb() string {
	return "Result"
}

func (m *Model) KeyHints() []ui.KeyHint {
	return []ui.KeyHint{
		{Key: "↑↓", Desc: "Navigate"},
		{Key: "Enter", Desc: "Select"},
		{Key: "Esc", Desc: "Home"},
	}
}
