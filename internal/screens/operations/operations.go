package operations

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dgr8akki/nano-ffmpeg/internal/screens"
	"github.com/dgr8akki/nano-ffmpeg/internal/ui"
)

// OperationID identifies an operation type.
type OperationID int

const (
	OpConvert OperationID = iota
	OpExtractAudio
	OpResize
	OpTrim
	OpCompress
	OpMerge
	OpSubtitles
	OpGIF
	OpThumbnails
	OpWatermark
	OpAudio
	OpFilters
)

// Operation describes a selectable operation.
type Operation struct {
	ID   OperationID
	Name string
	Desc string
	Icon string
}

// OperationSelectedMsg is sent when user picks an operation.
type OperationSelectedMsg struct {
	Operation Operation
}

var AllOperations = []Operation{
	{ID: OpConvert, Name: "Convert Format", Desc: "Change container or codec (MP4, MKV, WebM, MP3...)", Icon: ">>"},
	{ID: OpExtractAudio, Name: "Extract Audio", Desc: "Strip video track, keep audio", Icon: "♪ "},
	{ID: OpResize, Name: "Resize / Scale", Desc: "Change resolution, keep aspect ratio", Icon: "[]"},
	{ID: OpTrim, Name: "Trim / Cut", Desc: "Cut a segment by start/end time", Icon: "✂ "},
	{ID: OpCompress, Name: "Compress", Desc: "Reduce file size with quality control", Icon: "↓ "},
	{ID: OpMerge, Name: "Merge / Concat", Desc: "Join same-type files in the folder", Icon: "++"},
	{ID: OpSubtitles, Name: "Add Subtitles", Desc: "Burn in or re-mux the file's subtitle tracks", Icon: "T "},
	{ID: OpGIF, Name: "Create GIF", Desc: "Animated GIF from a video clip", Icon: "◎ "},
	{ID: OpThumbnails, Name: "Extract Thumbnails", Desc: "Grab frames as images", Icon: "▣ "},
	{ID: OpWatermark, Name: "Watermark", Desc: "Semi-transparent box overlay", Icon: "✦ "},
	{ID: OpAudio, Name: "Audio Adjustments", Desc: "Normalize, volume, fade in/out", Icon: "♫ "},
	{ID: OpFilters, Name: "Video Filters", Desc: "Stabilize, deinterlace, speed, rotate, flip", Icon: "◈ "},
}

// Model is the operations screen model.
type Model struct {
	cursor int
	width  int
	height int
}

// New creates a new operations screen.
func New() *Model {
	return &Model{}
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
			if m.cursor < len(AllOperations)-1 {
				m.cursor++
			}
		case "enter":
			op := AllOperations[m.cursor]
			return m, func() tea.Msg {
				return OperationSelectedMsg{Operation: op}
			}
		case "esc":
			return m, func() tea.Msg { return screens.BackMsg{} }
		}
	}
	return m, nil
}

func (m *Model) View() string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Foreground(ui.ColorPrimary).
		Bold(true).
		PaddingLeft(1).
		Render("What would you like to do?")
	b.WriteString(title)
	b.WriteString("\n\n")

	for i, op := range AllOperations {
		icon := lipgloss.NewStyle().
			Foreground(ui.ColorSecondary).
			Render(op.Icon)

		if i == m.cursor {
			indicator := lipgloss.NewStyle().
				Foreground(ui.ColorPrimary).
				Bold(true).
				Render(" > ")
			name := ui.SelectedStyle.Render(op.Name)
			desc := lipgloss.NewStyle().
				Foreground(ui.ColorDim).
				Render("  " + op.Desc)
			b.WriteString(indicator + icon + " " + name + desc + "\n")
		} else {
			name := lipgloss.NewStyle().
				Foreground(ui.ColorText).
				Render(op.Name)
			desc := lipgloss.NewStyle().
				Foreground(ui.ColorMuted).
				Render("  " + op.Desc)
			b.WriteString("   " + icon + " " + name + desc + "\n")
		}
	}

	return b.String()
}

func (m *Model) Breadcrumb() string {
	return "Operations"
}

func (m *Model) KeyHints() []ui.KeyHint {
	return []ui.KeyHint{
		{Key: "↑↓", Desc: "Navigate"},
		{Key: "Enter", Desc: "Select"},
		{Key: "Esc", Desc: "Back"},
	}
}
