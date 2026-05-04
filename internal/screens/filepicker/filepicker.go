package filepicker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dgr8akki/nano-ffmpeg/internal/ffmpeg"
	"github.com/dgr8akki/nano-ffmpeg/internal/screens"
	"github.com/dgr8akki/nano-ffmpeg/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var mediaExtensions = map[string]bool{
	".mp4": true, ".mkv": true, ".avi": true, ".mov": true, ".wmv": true,
	".flv": true, ".webm": true, ".m4v": true, ".mpg": true, ".mpeg": true,
	".3gp": true, ".ogv": true, ".ts": true, ".m2ts": true, ".vob": true,
	".mp3": true, ".aac": true, ".flac": true, ".wav": true, ".ogg": true,
	".wma": true, ".m4a": true, ".opus": true, ".ac3": true, ".dts": true,
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true,
	".webp": true, ".tiff": true, ".srt": true, ".ass": true, ".ssa": true,
}

type entry struct {
	name  string
	path  string
	isDir bool
	size  int64
}

// FileSelectedMsg is sent when a file is chosen.
type FileSelectedMsg struct {
	Path        string
	ProbeResult *ffmpeg.ProbeResult
}

// Model is the file picker screen model.
type Model struct {
	ffprobePath string
	currentDir  string
	entries     []entry
	cursor      int
	offset      int // scroll offset
	err         error
	probeResult *ffmpeg.ProbeResult
	pathInput   bool
	pathText    string
	width       int
	height      int
}

// New creates a new file picker starting at the given directory.
func New(ffprobePath string, startDir string) *Model {
	if startDir == "" {
		startDir, _ = os.UserHomeDir()
	}

	m := &Model{
		ffprobePath: ffprobePath,
		currentDir:  startDir,
	}
	m.loadDir()
	return m
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
		if m.pathInput {
			return m.updatePathInput(msg)
		}
		return m.updateBrowser(msg)
	}
	return m, nil
}

func (m *Model) updateBrowser(msg tea.KeyMsg) (screens.Screen, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.ensureVisible()
			m.probeSelected()
		}
	case "down", "j":
		if m.cursor < len(m.entries)-1 {
			m.cursor++
			m.ensureVisible()
			m.probeSelected()
		}
	case "enter":
		if m.cursor < len(m.entries) {
			e := m.entries[m.cursor]
			if e.isDir {
				m.currentDir = e.path
				m.cursor = 0
				m.offset = 0
				m.loadDir()
				m.probeResult = nil
			} else {
				// File selected -- probe and return message
				if m.probeResult != nil {
					return m, func() tea.Msg {
						return FileSelectedMsg{
							Path:        e.path,
							ProbeResult: m.probeResult,
						}
					}
				}
			}
		}
	case "esc":
		return m, func() tea.Msg { return screens.BackMsg{} }
	case "/":
		m.pathInput = true
		m.pathText = m.currentDir + "/"
	case "backspace":
		// Go to parent directory
		parent := filepath.Dir(m.currentDir)
		if parent != m.currentDir {
			m.currentDir = parent
			m.cursor = 0
			m.offset = 0
			m.loadDir()
			m.probeResult = nil
		}
	}
	return m, nil
}

func (m *Model) updatePathInput(msg tea.KeyMsg) (screens.Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.pathInput = false
	case "enter":
		m.pathInput = false
		info, err := os.Stat(m.pathText)
		if err != nil {
			m.err = err
			return m, nil
		}
		if info.IsDir() {
			m.currentDir = m.pathText
			m.cursor = 0
			m.offset = 0
			m.loadDir()
		} else {
			// It's a file -- probe it
			probe, err := ffmpeg.Probe(m.ffprobePath, m.pathText)
			if err == nil {
				return m, func() tea.Msg {
					return FileSelectedMsg{
						Path:        m.pathText,
						ProbeResult: probe,
					}
				}
			}
			m.err = err
		}
	case "backspace":
		if len(m.pathText) > 0 {
			m.pathText = m.pathText[:len(m.pathText)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.pathText += msg.String()
		}
	}
	return m, nil
}

func (m *Model) View() string {
	var b strings.Builder

	// Current directory header
	dirStyle := lipgloss.NewStyle().
		Foreground(ui.ColorSecondary).
		Bold(true).
		PaddingLeft(1)
	b.WriteString(dirStyle.Render(m.currentDir))
	b.WriteString("\n")

	if m.pathInput {
		b.WriteString(lipgloss.NewStyle().Foreground(ui.ColorPrimary).Render("  Path: "))
		b.WriteString(lipgloss.NewStyle().Foreground(ui.ColorText).Render(m.pathText))
		b.WriteString(lipgloss.NewStyle().Foreground(ui.ColorPrimary).Render("_"))
		b.WriteString("\n\n")
	}

	if m.err != nil {
		b.WriteString(ui.ErrorStyle.Render("  Error: " + m.err.Error()))
		b.WriteString("\n\n")
		m.err = nil
	}

	// File list
	visibleHeight := m.visibleLines()
	end := m.offset + visibleHeight
	if end > len(m.entries) {
		end = len(m.entries)
	}

	for i := m.offset; i < end; i++ {
		e := m.entries[i]
		selected := i == m.cursor

		icon := "  "
		if e.isDir {
			icon = "  "
		}

		name := e.name
		if e.isDir {
			name += "/"
		}

		var sizeStr string
		if !e.isDir {
			sizeStr = formatSize(e.size)
		}

		if selected {
			indicator := lipgloss.NewStyle().
				Foreground(ui.ColorPrimary).Bold(true).Render(" >")
			nameStyled := lipgloss.NewStyle().
				Foreground(ui.ColorText).Bold(true).Render(icon + name)
			sizeStyled := lipgloss.NewStyle().
				Foreground(ui.ColorDim).Render("  " + sizeStr)
			b.WriteString(indicator + nameStyled + sizeStyled + "\n")
		} else {
			nameColor := ui.ColorDim
			if e.isDir {
				nameColor = ui.ColorSecondary
			} else if isMediaFile(e.name) {
				nameColor = ui.ColorText
			}
			nameStyled := lipgloss.NewStyle().
				Foreground(nameColor).PaddingLeft(2).Render(icon + name)
			sizeStyled := lipgloss.NewStyle().
				Foreground(ui.ColorMuted).Render("  " + sizeStr)
			b.WriteString(nameStyled + sizeStyled + "\n")
		}
	}

	// Metadata preview panel
	if m.probeResult != nil {
		b.WriteString("\n")
		b.WriteString(m.renderPreview())
	}

	return b.String()
}

func (m *Model) renderPreview() string {
	r := m.probeResult
	var lines []string

	lines = append(lines, lipgloss.NewStyle().
		Foreground(ui.ColorPrimary).Bold(true).
		Render("File Info"))

	lines = append(lines, fmt.Sprintf("  Format: %s  |  Duration: %s  |  Size: %s",
		r.Format.FormatName, r.DurationString(), r.SizeString()))

	if v := r.VideoStream(); v != nil {
		fps := parseFPS(v.RFrameRate)
		line := fmt.Sprintf("  Video:  %s %dx%d", v.CodecName, v.Width, v.Height)
		if fps > 0 {
			line += fmt.Sprintf(" @ %.3gfps", fps)
		}
		if v.PixFmt != "" {
			line += fmt.Sprintf(" (%s)", v.PixFmt)
		}
		lines = append(lines, line)
	}

	if a := r.AudioStream(); a != nil {
		line := fmt.Sprintf("  Audio:  %s", a.CodecName)
		if a.ChannelLayout != "" {
			line += " " + a.ChannelLayout
		}
		if a.SampleRate != "" {
			line += fmt.Sprintf(" %sHz", a.SampleRate)
		}
		lines = append(lines, line)
	}

	subs := r.SubtitleStreams()
	if len(subs) > 0 {
		line := fmt.Sprintf("  Subs:   %d track(s)", len(subs))
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	return ui.PanelStyle.Render(
		lipgloss.NewStyle().Foreground(ui.ColorDim).Render(content),
	)
}

func (m *Model) Breadcrumb() string {
	return "File Picker"
}

func (m *Model) KeyHints() []ui.KeyHint {
	if m.pathInput {
		return []ui.KeyHint{
			{Key: "Enter", Desc: "Go"},
			{Key: "Esc", Desc: "Cancel"},
		}
	}
	return []ui.KeyHint{
		{Key: "↑↓", Desc: "Navigate"},
		{Key: "Enter", Desc: "Open/Select"},
		{Key: "Bksp", Desc: "Parent dir"},
		{Key: "/", Desc: "Path input"},
		{Key: "Esc", Desc: "Back"},
	}
}

func (m *Model) loadDir() {
	raw, err := os.ReadDir(m.currentDir)
	if err != nil {
		m.err = err
		return
	}

	m.entries = nil

	type resolved struct {
		name  string
		path  string
		isDir bool
		size  int64
	}
	var items []resolved
	for _, e := range raw {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		path := filepath.Join(m.currentDir, e.Name())
		// Stat follows symlinks so directory symlinks are browsable; broken
		// symlinks return an error and are skipped.
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		items = append(items, resolved{
			name:  e.Name(),
			path:  path,
			isDir: info.IsDir(),
			size:  info.Size(),
		})
	}

	for _, it := range items {
		if it.isDir {
			m.entries = append(m.entries, entry{name: it.name, path: it.path, isDir: true})
		}
	}
	for _, it := range items {
		if !it.isDir {
			m.entries = append(m.entries, entry{name: it.name, path: it.path, isDir: false, size: it.size})
		}
	}
}

func (m *Model) probeSelected() {
	if m.cursor >= len(m.entries) {
		return
	}
	e := m.entries[m.cursor]
	if e.isDir || !isMediaFile(e.name) {
		m.probeResult = nil
		return
	}
	result, err := ffmpeg.Probe(m.ffprobePath, e.path)
	if err != nil {
		m.probeResult = nil
		return
	}
	m.probeResult = result
}

func (m *Model) visibleLines() int {
	h := m.height - 10 // reserve for header, preview, chrome
	if h < 5 {
		h = 5
	}
	return h
}

func (m *Model) ensureVisible() {
	visible := m.visibleLines()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
}

func isMediaFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return mediaExtensions[ext]
}

func formatSize(bytes int64) string {
	size := float64(bytes)
	units := []string{"B", "KB", "MB", "GB"}
	i := 0
	for size >= 1024 && i < len(units)-1 {
		size /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%.0f %s", size, units[i])
	}
	return fmt.Sprintf("%.1f %s", size, units[i])
}

func parseFPS(rational string) float64 {
	parts := strings.Split(rational, "/")
	if len(parts) != 2 {
		return 0
	}
	num := 0.0
	den := 0.0
	fmt.Sscanf(parts[0], "%f", &num)
	fmt.Sscanf(parts[1], "%f", &den)
	if den == 0 {
		return 0
	}
	return num / den
}
