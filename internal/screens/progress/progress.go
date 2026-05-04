package progress

import (
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dgr8akki/nano-ffmpeg/internal/ffmpeg"
	"github.com/dgr8akki/nano-ffmpeg/internal/screens"
	"github.com/dgr8akki/nano-ffmpeg/internal/ui"
)

// DoneMsg signals encoding complete.
type DoneMsg struct {
	Err        error
	OutputPath string
	InputSize  int64
}

type tickMsg time.Time

// lineBuffer collects stderr lines from the background goroutine.
type lineBuffer struct {
	mu    sync.Mutex
	lines []string
}

func (lb *lineBuffer) add(line string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.lines = append(lb.lines, line)
}

func (lb *lineBuffer) drain() []string {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	out := lb.lines
	lb.lines = nil
	return out
}

// Model is the progress screen model.
type Model struct {
	commands     []*ffmpeg.Command
	runner       *ffmpeg.Runner
	runnerMu     sync.Mutex
	parser       *ffmpeg.ProgressParser
	progress     *ffmpeg.Progress
	buf          *lineBuffer
	logLines     []string
	maxLogLines  int
	inputFile    string
	outputFile   string
	inputSize    int64
	done         bool
	err          error
	canceling    bool
	width        int
	height       int
	spinnerFrame int
	spinnerChars []string
}

// New creates a new progress screen.
func New(commands []*ffmpeg.Command, totalDuration float64, inputSize int64) *Model {

	m := &Model{
		commands:     commands,
		parser:       ffmpeg.NewProgressParser(totalDuration),
		buf:          &lineBuffer{},
		logLines:     make([]string, 0, 100),
		maxLogLines:  50,
		inputSize:    inputSize,
		spinnerChars: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	}
	if len(commands) == 0 {
		m.err = fmt.Errorf("no ffmpeg command to execute")
		return m
	}
	m.inputFile = commands[0].Input
	m.outputFile = commands[len(commands)-1].Output
	return m
}

func (m *Model) Init() tea.Cmd {
	if m.err != nil {
		return nil
	}
	return tea.Batch(m.startRunner(), m.tick())
}

func (m *Model) startRunner() tea.Cmd {
	return func() tea.Msg {
		if len(m.commands) == 0 {
			return DoneMsg{Err: fmt.Errorf("no ffmpeg command to execute")}
		}

		defer m.cleanupTempFiles()

		for i, cmd := range m.commands {
			runner, err := ffmpeg.NewRunner(cmd)
			if err != nil {
				return DoneMsg{Err: err}
			}
			m.setRunner(runner)

			if len(m.commands) > 1 {
				m.buf.add(fmt.Sprintf("pass %d/%d started", i+1, len(m.commands)))
			}

			if err := runner.Start(); err != nil {
				return DoneMsg{Err: err}
			}

			// Read stderr in this goroutine, buffer lines for tick to consume
			scanner := runner.ScanStderr()
			for scanner.Scan() {
				line := scanner.Text()
				if line != "" {
					m.buf.add(line)
				}
			}
			if err := scanner.Err(); err != nil {
				return DoneMsg{Err: err}
			}

			if err := runner.Wait(); err != nil {
				return DoneMsg{
					Err:        err,
					OutputPath: m.outputFile,
					InputSize:  m.inputSize,
				}
			}

			if len(m.commands) > 1 && i < len(m.commands)-1 {
				m.buf.add(fmt.Sprintf("pass %d/%d complete", i+1, len(m.commands)))
			}
		}

		m.setRunner(nil)
		return DoneMsg{
			Err:        nil,
			OutputPath: m.outputFile,
			InputSize:  m.inputSize,
		}
	}
}

func (m *Model) cleanupTempFiles() {
	seen := map[string]struct{}{}
	for _, cmd := range m.commands {
		for _, p := range cmd.Cleanup {
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			os.Remove(p)
		}
	}
}

func (m *Model) setRunner(r *ffmpeg.Runner) {
	m.runnerMu.Lock()
	defer m.runnerMu.Unlock()
	m.runner = r
}

func (m *Model) activeRunner() *ffmpeg.Runner {
	m.runnerMu.Lock()
	defer m.runnerMu.Unlock()
	return m.runner
}

func (m *Model) tick() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		m.spinnerFrame = (m.spinnerFrame + 1) % len(m.spinnerChars)
		if !m.done {
			// Drain buffered lines from the background reader
			for _, line := range m.buf.drain() {
				m.addLogLine(line)
				if p := m.parser.Parse(line); p != nil {
					m.progress = p
				}
			}
			return m, m.tick()
		}

	case DoneMsg:
		// Drain any remaining lines
		for _, line := range m.buf.drain() {
			m.addLogLine(line)
			if p := m.parser.Parse(line); p != nil {
				m.progress = p
			}
		}
		m.done = true
		m.err = msg.Err
		if m.err == nil {
			return m, func() tea.Msg {
				return DoneMsg{OutputPath: msg.OutputPath, InputSize: msg.InputSize}
			}
		}

	case tea.KeyMsg:
		if m.canceling {
			switch msg.String() {
			case "y":
				if runner := m.activeRunner(); runner != nil {
					runner.Cancel()
					runner.CleanupOutput()
				}
				m.done = true
				return m, func() tea.Msg { return screens.BackMsg{} }
			case "n", "esc":
				m.canceling = false
			}
		} else {
			switch msg.String() {
			case "esc":
				m.canceling = true
			}
		}
	}
	return m, nil
}

func (m *Model) addLogLine(line string) {
	m.logLines = append(m.logLines, line)
	if len(m.logLines) > m.maxLogLines {
		m.logLines = m.logLines[1:]
	}
}

func (m *Model) View() string {
	var b strings.Builder

	if m.err != nil && m.done {
		b.WriteString(ui.ErrorStyle.Render("  Encoding failed: " + m.err.Error()))
		if detail := m.errorDetail(); detail != "" {
			b.WriteString("\n")
			b.WriteString(ui.MutedStyle.Render("  ffmpeg: " + detail))
		}
		b.WriteString("\n\n")
		b.WriteString(ui.MutedStyle.Render("  Press Esc to go back"))
		return b.String()
	}

	// File info
	b.WriteString(m.renderFileInfo())
	b.WriteString("\n\n")

	// Progress bar
	b.WriteString(m.renderProgressBar())
	b.WriteString("\n\n")

	// Stats grid
	b.WriteString(m.renderStats())
	b.WriteString("\n\n")

	// Cancel confirmation
	if m.canceling {
		b.WriteString(ui.WarningStyle.Render("  Cancel encoding? "))
		b.WriteString(ui.KeyStyle.Render("[y]"))
		b.WriteString(ui.MutedStyle.Render(" Yes  "))
		b.WriteString(ui.KeyStyle.Render("[n]"))
		b.WriteString(ui.MutedStyle.Render(" No"))
		b.WriteString("\n\n")
	}

	// Log panel
	b.WriteString(m.renderLog())

	return b.String()
}

func (m *Model) renderFileInfo() string {
	arrow := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).Render(" --> ")
	input := lipgloss.NewStyle().Foreground(ui.ColorText).Render("  " + shortPath(m.inputFile))
	output := lipgloss.NewStyle().Foreground(ui.ColorSecondary).Render(shortPath(m.outputFile))
	return input + arrow + output
}

func (m *Model) renderProgressBar() string {
	width := m.width - 16
	if width < 20 {
		width = 20
	}

	percent := 0.0
	if m.progress != nil {
		percent = m.progress.Percent
	}

	filled := int(math.Round(float64(width) * percent / 100))
	if filled > width {
		filled = width
	}
	empty := width - filled

	// Gradient bar: green to cyan
	var bar strings.Builder
	for i := 0; i < filled; i++ {
		ratio := float64(i) / float64(width)
		if ratio < 0.5 {
			bar.WriteString(lipgloss.NewStyle().Foreground(ui.ColorSuccess).Render("█"))
		} else {
			bar.WriteString(lipgloss.NewStyle().Foreground(ui.ColorSecondary).Render("█"))
		}
	}
	bar.WriteString(lipgloss.NewStyle().Foreground(ui.ColorProgressEmpty).Render(strings.Repeat("░", empty)))

	percentStr := fmt.Sprintf(" %5.1f%%", percent)
	percentStyled := lipgloss.NewStyle().Foreground(ui.ColorText).Bold(true).Render(percentStr)

	if percent <= 0 && !m.done {
		spinner := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).Render(
			"  " + m.spinnerChars[m.spinnerFrame] + " Encoding...")
		return spinner
	}

	return "  " + bar.String() + percentStyled
}

func (m *Model) renderStats() string {
	p := m.progress
	if p == nil {
		p = &ffmpeg.Progress{}
	}

	col1Style := lipgloss.NewStyle().Foreground(ui.ColorDim).Width(28).PaddingLeft(2)
	col2Style := lipgloss.NewStyle().Foreground(ui.ColorDim).Width(28)
	valStyle := lipgloss.NewStyle().Foreground(ui.ColorText).Bold(true)

	elapsed := ffmpeg.FormatDuration(p.Elapsed)
	eta := "--:--:--"
	if p.ETA > 0 {
		eta = ffmpeg.FormatDuration(p.ETA)
	}

	speed := "--"
	if p.Speed > 0 {
		speed = fmt.Sprintf("%.1fx", p.Speed)
	}

	frames := "--"
	if p.Frame > 0 {
		frames = fmt.Sprintf("%d", p.Frame)
	}

	size := "--"
	if p.Size > 0 {
		size = ffmpeg.FormatSize(p.Size)
	}

	bitrate := "--"
	if p.Bitrate > 0 {
		bitrate = fmt.Sprintf("%.0f kbps", p.Bitrate)
	}

	fps := "--"
	if p.FPS > 0 {
		fps = fmt.Sprintf("%.1f", p.FPS)
	}

	lines := []string{
		col1Style.Render("Elapsed   "+valStyle.Render(elapsed)) +
			col2Style.Render("Frames    "+valStyle.Render(frames)),
		col1Style.Render("ETA       "+valStyle.Render(eta)) +
			col2Style.Render("Size      "+valStyle.Render(size)),
		col1Style.Render("Speed     "+valStyle.Render(speed)) +
			col2Style.Render("Bitrate   "+valStyle.Render(bitrate)),
		col1Style.Render("FPS       " + valStyle.Render(fps)),
	}

	return strings.Join(lines, "\n")
}

func (m *Model) renderLog() string {
	title := lipgloss.NewStyle().
		Foreground(ui.ColorPrimary).
		Bold(true).
		Render("Live Log")

	visible := 6
	start := 0
	if len(m.logLines) > visible {
		start = len(m.logLines) - visible
	}

	var logContent strings.Builder
	for _, line := range m.logLines[start:] {
		truncated := line
		maxWidth := m.width - 8
		if maxWidth > 0 && len(truncated) > maxWidth {
			truncated = truncated[:maxWidth]
		}
		logContent.WriteString(lipgloss.NewStyle().Foreground(ui.ColorMuted).Render(truncated))
		logContent.WriteString("\n")
	}

	return ui.PanelStyle.Render(title + "\n" + logContent.String())
}

func (m *Model) Breadcrumb() string {
	return "Encoding"
}

func (m *Model) KeyHints() []ui.KeyHint {
	if m.canceling {
		return []ui.KeyHint{
			{Key: "y", Desc: "Confirm cancel"},
			{Key: "n", Desc: "Continue"},
		}
	}
	return []ui.KeyHint{
		{Key: "Esc", Desc: "Cancel"},
	}
}

func shortPath(path string) string {
	if len(path) > 50 {
		return "..." + path[len(path)-47:]
	}
	return path
}

func (m *Model) errorDetail() string {
	for i := len(m.logLines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(m.logLines[i])
		if line == "" {
			continue
		}

		lower := strings.ToLower(line)
		if strings.Contains(lower, "error") ||
			strings.Contains(lower, "invalid") ||
			strings.Contains(lower, "failed") ||
			strings.Contains(lower, "unable") ||
			strings.Contains(lower, "cannot") ||
			strings.Contains(lower, "not found") {
			return line
		}
	}

	if len(m.logLines) > 0 {
		return strings.TrimSpace(m.logLines[len(m.logLines)-1])
	}
	return ""
}
