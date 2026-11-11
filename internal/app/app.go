package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dgr8akki/nano-ffmpeg/internal/ffmpeg"
	"github.com/dgr8akki/nano-ffmpeg/internal/screens"
	"github.com/dgr8akki/nano-ffmpeg/internal/screens/filepicker"
	"github.com/dgr8akki/nano-ffmpeg/internal/screens/home"
	"github.com/dgr8akki/nano-ffmpeg/internal/screens/operations"
	"github.com/dgr8akki/nano-ffmpeg/internal/screens/progress"
	"github.com/dgr8akki/nano-ffmpeg/internal/screens/result"
	"github.com/dgr8akki/nano-ffmpeg/internal/screens/settings"
	"github.com/dgr8akki/nano-ffmpeg/internal/ui"
)

// Model is the top-level Bubble Tea model.
type Model struct {
	ffmpegInfo   *ffmpeg.Info
	caps         *ffmpeg.Capabilities
	config       *Config
	startDir     string
	screen       screens.Screen
	screenStack  []screens.Screen
	frame        *ui.Frame
	statusLine   string
	selectedFile *filepicker.FileSelectedMsg
	pendingOp    *operations.Operation // picked on Home before choosing a file
	showHelp     bool
	width        int
	height       int
}

// RunOptions controls runtime behavior for app startup.
type RunOptions struct {
	Theme       string
	StartDir    string
	InitialFile *InitialFile
}

// InitialFile is an optional startup file that skips file picker.
type InitialFile struct {
	Path        string
	ProbeResult *ffmpeg.ProbeResult
}

// New creates the top-level app model.
func New(info *ffmpeg.Info, caps *ffmpeg.Capabilities, opts RunOptions) Model {
	cfg := LoadConfig()
	theme := cfg.Theme
	if opts.Theme != "" {
		theme = opts.Theme
	}
	ui.SetTheme(theme)
	homeScreen := home.New(info, caps, cfg.RecentFiles)
	model := Model{
		ffmpegInfo: info,
		caps:       caps,
		config:     cfg,
		startDir:   opts.StartDir,
		screen:     homeScreen,
		frame:      ui.NewFrame(80, 24),
	}

	if opts.InitialFile != nil && opts.InitialFile.Path != "" && opts.InitialFile.ProbeResult != nil {
		model.selectedFile = &filepicker.FileSelectedMsg{
			Path:        opts.InitialFile.Path,
			ProbeResult: opts.InitialFile.ProbeResult,
		}
		model.statusLine = opts.InitialFile.ProbeResult.StatusLine()
		model.config.AddRecentFile(opts.InitialFile.Path)
		model.screen = operations.New()
		model.screenStack = []screens.Screen{homeScreen}
	}

	return model
}

func (m Model) Init() tea.Cmd {
	return m.screen.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.frame = ui.NewFrame(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m.quit()
		}
		// While a text field has focus, q and ? are just characters.
		if tc, ok := m.screen.(interface{ CapturesText() bool }); ok && tc.CapturesText() && !m.showHelp {
			break
		}
		// Help toggle intercepts everything
		if msg.String() == "?" {
			m.showHelp = !m.showHelp
			return m, nil
		}
		if m.showHelp {
			if msg.String() == "esc" {
				m.showHelp = false
			}
			return m, nil
		}
		if msg.String() == "q" {
			return m.quit()
		}

	case screens.NavigateMsg:
		if msg.Screen == screens.ScreenHome {
			m.screenStack = nil
			m.screen = home.New(m.ffmpegInfo, m.caps, m.config.RecentFiles)
			m.statusLine = ""
			m.selectedFile = nil
			m.pendingOp = nil
			return m, m.screen.Init()
		}
		if op, ok := msg.Payload.(operations.Operation); ok && msg.Screen == screens.ScreenFilePicker {
			m.pendingOp = &op
		}
		m.pushScreen(m.resolveScreen(msg))
		return m, m.screen.Init()

	case filepicker.FileSelectedMsg:
		m.selectedFile = &msg
		m.statusLine = msg.ProbeResult.StatusLine()
		// Save to recent files
		m.config.AddRecentFile(msg.Path)
		_ = m.config.Save()
		m.pushScreen(operations.New())
		if op := m.pendingOp; op != nil {
			// Home already chose the operation; go straight to its settings.
			m.pendingOp = nil
			return m.Update(operations.OperationSelectedMsg{Operation: *op})
		}
		return m, m.screen.Init()

	case operations.OperationSelectedMsg:
		if m.selectedFile != nil {
			s := settings.New(
				msg.Operation.ID,
				msg.Operation.Name,
				m.selectedFile.Path,
				m.selectedFile.ProbeResult,
				m.ffmpegInfo.FFmpegPath,
			)
			if dir := m.config.DefaultOutputDir; dir != "" {
				s.SetOutputDir(dir)
			}
			m.pushScreen(s)
			return m, m.screen.Init()
		}

	case settings.ExecuteMsg:
		if m.selectedFile != nil {
			totalDuration := m.selectedFile.ProbeResult.Format.Duration
			if msg.Duration > 0 {
				totalDuration = msg.Duration
			}
			inputSize := m.selectedFile.ProbeResult.Format.Size
			ps := progress.New(msg.Commands, totalDuration, inputSize)
			m.pushScreen(ps)
			return m, m.screen.Init()
		}

	case progress.DoneMsg:
		if msg.Err == nil {
			rs := result.New(msg.OutputPath, msg.InputSize)
			m.screenStack = nil
			m.screen = rs
			return m, m.screen.Init()
		}

	case screens.BackMsg:
		if len(m.screenStack) > 0 {
			m.screen = m.screenStack[len(m.screenStack)-1]
			m.screenStack = m.screenStack[:len(m.screenStack)-1]
		}
		return m, nil

	case screens.StatusMsg:
		m.statusLine = msg.Text
		return m, nil
	}

	var cmd tea.Cmd
	m.screen, cmd = m.screen.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	// Check terminal size
	if warning := ui.CheckTerminalSize(m.width, m.height); warning != "" {
		return warning
	}

	if m.showHelp {
		helpSections := m.helpForCurrentScreen()
		return ui.HelpOverlay(helpSections, m.width, m.height)
	}

	content := m.screen.View()
	hints := m.screen.KeyHints()
	breadcrumb := m.screen.Breadcrumb()
	return m.frame.Render(breadcrumb, m.statusLine, content, hints)
}

// quit stops any running ffmpeg job (and removes its partial output) before
// exiting, so quitting never leaves an orphaned encode behind.
func (m Model) quit() (tea.Model, tea.Cmd) {
	if c, ok := m.screen.(interface{ Cancel() }); ok {
		c.Cancel()
	}
	return m, tea.Quit
}

func (m Model) helpForCurrentScreen() []ui.HelpSection {
	switch m.screen.Breadcrumb() {
	case "Home":
		return ui.HomeHelp()
	case "File Picker":
		return ui.FilePickerHelp()
	case "Operations":
		return ui.OperationsHelp()
	case "Encoding":
		return ui.ProgressHelp()
	default:
		return ui.SettingsHelp()
	}
}

func (m *Model) pushScreen(s screens.Screen) {
	m.screenStack = append(m.screenStack, m.screen)
	m.screen = s
}

func (m Model) resolveScreen(msg screens.NavigateMsg) screens.Screen {
	switch msg.Screen {
	case screens.ScreenHome:
		return home.New(m.ffmpegInfo, m.caps, m.config.RecentFiles)
	case screens.ScreenFilePicker:
		return filepicker.New(m.ffmpegInfo.FFprobePath, m.startDir)
	case screens.ScreenOperations:
		return operations.New()
	default:
		return home.New(m.ffmpegInfo, m.caps, m.config.RecentFiles)
	}
}

// Run starts the TUI application.
func Run(info *ffmpeg.Info, opts RunOptions) error {
	caps, err := ffmpeg.ProbeCapabilities(info)
	if err != nil {
		return fmt.Errorf("failed to probe ffmpeg capabilities: %w", err)
	}
	model := New(info, caps, opts)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
