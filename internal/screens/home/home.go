package home

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dgr8akki/nano-ffmpeg/internal/ffmpeg"
	"github.com/dgr8akki/nano-ffmpeg/internal/screens"
	"github.com/dgr8akki/nano-ffmpeg/internal/screens/operations"
	"github.com/dgr8akki/nano-ffmpeg/internal/ui"
)

// Model is the home screen model.
type Model struct {
	info        *ffmpeg.Info
	caps        *ffmpeg.Capabilities
	recentFiles []string
	cursor      int
	width       int
	height      int
}

// New creates a new home screen.
func New(info *ffmpeg.Info, caps *ffmpeg.Capabilities, recentFiles []string) *Model {
	return &Model{
		info:        info,
		caps:        caps,
		recentFiles: recentFiles,
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
			if m.cursor < len(operations.AllOperations)-1 {
				m.cursor++
			}
		case "enter":
			return m, func() tea.Msg {
				return screens.NavigateMsg{
					Screen:  screens.ScreenFilePicker,
					Payload: operations.AllOperations[m.cursor],
				}
			}
		}
	}

	return m, nil
}

func (m *Model) View() string {
	var b strings.Builder

	// Header section
	header := m.renderHeader()
	b.WriteString(header)
	b.WriteString("\n\n")

	// Recent files
	if len(m.recentFiles) > 0 {
		b.WriteString(ui.SubtitleStyle.Render("RECENT FILES"))
		b.WriteString("\n")
		for i, f := range m.recentFiles {
			if i >= 5 {
				break
			}
			name := filepath.Base(f)
			dir := filepath.Dir(f)
			nameStyled := lipgloss.NewStyle().Foreground(ui.ColorText).PaddingLeft(3).Render(name)
			dirStyled := lipgloss.NewStyle().Foreground(ui.ColorMuted).Render("  " + dir)
			b.WriteString(nameStyled + dirStyled + "\n")
		}
		b.WriteString("\n")
	}

	// Operations list
	b.WriteString(ui.SubtitleStyle.Render("OPERATIONS"))
	b.WriteString("\n\n")

	for i, op := range operations.AllOperations {
		if i == m.cursor {
			indicator := lipgloss.NewStyle().
				Foreground(ui.ColorPrimary).
				Bold(true).
				Render(" > ")
			name := ui.SelectedStyle.Render(op.Name)
			desc := lipgloss.NewStyle().
				Foreground(ui.ColorDim).
				Render("  " + op.Desc)
			b.WriteString(indicator + name + desc + "\n")
		} else {
			name := lipgloss.NewStyle().
				Foreground(ui.ColorText).
				PaddingLeft(3).
				Render(op.Name)
			desc := lipgloss.NewStyle().
				Foreground(ui.ColorMuted).
				Render("  " + op.Desc)
			b.WriteString(name + desc + "\n")
		}
	}

	return b.String()
}

func (m *Model) renderHeader() string {
	versionLine := fmt.Sprintf("ffmpeg %s", m.info.Version)
	versionStyled := ui.SuccessStyle.Render("  " + versionLine)

	codecCount := 0
	encoderCount := 0
	for _, c := range m.caps.Codecs {
		codecCount++
		if c.Encoding {
			encoderCount++
		}
	}

	statsLine := fmt.Sprintf(
		"%d codecs  |  %d encoders  |  %d formats  |  %d filters",
		codecCount,
		encoderCount,
		len(m.caps.Formats),
		len(m.caps.Filters),
	)

	hwLine := ""
	if len(m.caps.HWAccels) > 0 {
		hwLine = fmt.Sprintf("HW Accel: %s", strings.Join(m.caps.HWAccels, ", "))
		hwLine = lipgloss.NewStyle().Foreground(ui.ColorSecondary).Render("  " + hwLine)
	}

	info := versionStyled + "\n" +
		lipgloss.NewStyle().Foreground(ui.ColorDim).PaddingLeft(2).Render(statsLine)
	if hwLine != "" {
		info += "\n" + hwLine
	}

	return ui.PanelStyle.Render(info)
}

func (m *Model) Breadcrumb() string {
	return "Home"
}

func (m *Model) KeyHints() []ui.KeyHint {
	return []ui.KeyHint{
		{Key: "↑↓", Desc: "Navigate"},
		{Key: "Enter", Desc: "Select"},
		{Key: "q", Desc: "Quit"},
		{Key: "?", Desc: "Help"},
	}
}
