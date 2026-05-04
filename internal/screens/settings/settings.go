package settings

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dgr8akki/nano-ffmpeg/internal/ffmpeg"
	"github.com/dgr8akki/nano-ffmpeg/internal/screens"
	"github.com/dgr8akki/nano-ffmpeg/internal/screens/operations"
	"github.com/dgr8akki/nano-ffmpeg/internal/ui"
)

// FieldType defines the kind of form field.
type FieldType int

const (
	FieldSelect FieldType = iota
	FieldText
	FieldToggle
)

// Option is a selectable choice in a select field.
type Option struct {
	Label string
	Value string
}

// Field defines a form field.
type Field struct {
	Label    string
	Type     FieldType
	Options  []Option // for FieldSelect
	Value    string   // current value
	Selected int      // selected index for FieldSelect
	Enabled  bool     // for FieldToggle
	Cursor   int      // cursor position for FieldText
}

// ExecuteMsg tells the app to run the ffmpeg command.
type ExecuteMsg struct {
	Commands []*ffmpeg.Command
}

// Model is the settings screen model.
type Model struct {
	opID         operations.OperationID
	opName       string
	fields       []Field
	cursor       int
	filePath     string
	outputDir    string
	probeResult  *ffmpeg.ProbeResult
	ffmpegPath   string
	vidstabOK    bool
	vidstabKnown bool
	width        int
	height       int
}

// New creates a settings screen for the given operation and input file.
func New(opID operations.OperationID, opName string, filePath string, probe *ffmpeg.ProbeResult, ffmpegPath string) *Model {
	m := &Model{
		opID:        opID,
		opName:      opName,
		filePath:    filePath,
		outputDir:   filepath.Dir(filePath),
		probeResult: probe,
		ffmpegPath:  ffmpegPath,
	}
	m.fields = m.buildFields()
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
		if m.handleTextInput(msg) {
			return m, nil
		}
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.fields)-1 {
				m.cursor++
			}
		case "left", "h":
			m.adjustField(-1)
		case "right", "l":
			m.adjustField(1)
		case "enter":
			commands := m.buildCommands()
			return m, func() tea.Msg {
				return ExecuteMsg{Commands: commands}
			}
		case "esc":
			return m, func() tea.Msg { return screens.BackMsg{} }
		case "c":
			// Copy command to clipboard (handled by app)
		}
	}
	return m, nil
}

func (m *Model) adjustField(delta int) {
	if m.cursor >= len(m.fields) {
		return
	}
	f := &m.fields[m.cursor]
	switch f.Type {
	case FieldSelect:
		f.Selected += delta
		if f.Selected < 0 {
			f.Selected = 0
		}
		if f.Selected >= len(f.Options) {
			f.Selected = len(f.Options) - 1
		}
		f.Value = f.Options[f.Selected].Value
	case FieldToggle:
		f.Enabled = !f.Enabled
		if f.Enabled {
			f.Value = "true"
		} else {
			f.Value = "false"
		}
	case FieldText:
		f.Cursor += delta
		if f.Cursor < 0 {
			f.Cursor = 0
		}
		if f.Cursor > len([]rune(f.Value)) {
			f.Cursor = len([]rune(f.Value))
		}
	}
	m.reconcileCompressFields()
}

func (m *Model) reconcileCompressFields() {
	if m.opID != operations.OpCompress {
		return
	}
	twoPass := m.fieldEnabled("Two-Pass")
	codec := m.fieldValue("Codec")
	if twoPass && !ffmpeg.TwoPassSupported(codec) {
		twoPass = false
		for i := range m.fields {
			if m.fields[i].Label == "Two-Pass" {
				m.fields[i].Enabled = false
				m.fields[i].Value = "false"
			}
		}
	}

	hasBitrate := false
	hasQuality := false
	for _, f := range m.fields {
		switch f.Label {
		case "Target Bitrate":
			hasBitrate = true
		case "Quality":
			hasQuality = true
		}
	}
	if twoPass == hasBitrate && twoPass != hasQuality {
		return
	}

	quality := m.fieldValue("Quality")
	if quality == "" {
		quality = "23"
	}
	bitrate := m.fieldValue("Target Bitrate")
	if bitrate == "" {
		bitrate = "2500k"
	}

	preserved := map[string]Field{}
	for _, f := range m.fields {
		preserved[f.Label] = f
	}
	rebuilt := m.compressFieldsWith(twoPass, quality, bitrate)
	for i, f := range rebuilt {
		if prev, ok := preserved[f.Label]; ok && f.Label != "Quality" && f.Label != "Target Bitrate" && f.Label != "Two-Pass" {
			rebuilt[i] = prev
		}
	}
	m.fields = rebuilt
	if m.cursor >= len(m.fields) {
		m.cursor = len(m.fields) - 1
	}
}

func (m *Model) handleTextInput(msg tea.KeyMsg) bool {
	if m.cursor < 0 || m.cursor >= len(m.fields) {
		return false
	}
	f := &m.fields[m.cursor]
	if f.Type != FieldText {
		return false
	}

	switch msg.Type {
	case tea.KeyRunes:
		if len(msg.Runes) == 0 {
			return false
		}
		m.insertText(f, string(msg.Runes))
		return true
	case tea.KeyBackspace:
		return m.backspaceText(f)
	case tea.KeyDelete:
		return m.deleteText(f)
	case tea.KeyHome:
		f.Cursor = 0
		return true
	case tea.KeyEnd:
		f.Cursor = len([]rune(f.Value))
		return true
	}

	switch msg.String() {
	case "ctrl+h":
		return m.backspaceText(f)
	}

	return false
}

func (m *Model) insertText(f *Field, s string) {
	runes := []rune(f.Value)
	cursor := clampCursor(f.Cursor, len(runes))
	insert := []rune(s)

	runes = append(runes[:cursor], append(insert, runes[cursor:]...)...)
	f.Value = string(runes)
	f.Cursor = cursor + len(insert)
}

func (m *Model) backspaceText(f *Field) bool {
	runes := []rune(f.Value)
	cursor := clampCursor(f.Cursor, len(runes))
	if cursor == 0 {
		f.Cursor = 0
		return false
	}

	runes = append(runes[:cursor-1], runes[cursor:]...)
	f.Value = string(runes)
	f.Cursor = cursor - 1
	return true
}

func (m *Model) deleteText(f *Field) bool {
	runes := []rune(f.Value)
	cursor := clampCursor(f.Cursor, len(runes))
	if cursor >= len(runes) {
		f.Cursor = len(runes)
		return false
	}

	runes = append(runes[:cursor], runes[cursor+1:]...)
	f.Value = string(runes)
	f.Cursor = cursor
	return true
}

func clampCursor(cursor, max int) int {
	if cursor < 0 {
		return 0
	}
	if cursor > max {
		return max
	}
	return cursor
}

func (m *Model) View() string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Foreground(ui.ColorPrimary).
		Bold(true).
		PaddingLeft(1).
		Render(m.opName + " Settings")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Form fields
	for i, f := range m.fields {
		selected := i == m.cursor
		b.WriteString(m.renderField(f, selected))
		b.WriteString("\n")
	}
	if notice := m.fallbackNotice(); notice != "" {
		b.WriteString("\n")
		b.WriteString(ui.WarningStyle.Render("  " + notice))
		b.WriteString("\n")
	}

	// Output info
	b.WriteString("\n")
	outLabel := lipgloss.NewStyle().Foreground(ui.ColorDim).Render("  Output: ")
	outPath := lipgloss.NewStyle().Foreground(ui.ColorSecondary).Render(m.outputPath())
	b.WriteString(outLabel + outPath + "\n")

	// Command preview
	b.WriteString("\n")
	cmdPreview := lipgloss.NewStyle().
		Foreground(ui.ColorDim).
		PaddingLeft(2).
		Render("$ " + m.commandPreview())
	previewBox := ui.PanelStyle.Render(
		lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).Render("Command Preview") +
			"\n" + cmdPreview,
	)
	b.WriteString(previewBox)

	return b.String()
}

func (m *Model) renderField(f Field, selected bool) string {
	indicator := "  "
	if selected {
		indicator = lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).Render("> ")
	}

	label := lipgloss.NewStyle().
		Foreground(ui.ColorText).
		Width(20).
		Render(f.Label)

	var value string
	switch f.Type {
	case FieldSelect:
		var parts []string
		for i, opt := range f.Options {
			if i == f.Selected {
				parts = append(parts, lipgloss.NewStyle().
					Foreground(ui.ColorText).
					Background(ui.ColorHighlight).
					Bold(true).
					Padding(0, 1).
					Render(opt.Label))
			} else {
				parts = append(parts, lipgloss.NewStyle().
					Foreground(ui.ColorMuted).
					Padding(0, 1).
					Render(opt.Label))
			}
		}
		value = strings.Join(parts, " ")
	case FieldToggle:
		if f.Enabled {
			value = lipgloss.NewStyle().Foreground(ui.ColorSuccess).Bold(true).Render("[ON]")
		} else {
			value = lipgloss.NewStyle().Foreground(ui.ColorMuted).Render("[OFF]")
		}
	case FieldText:
		display := f.Value
		if selected {
			display = textWithCursor(f.Value, f.Cursor)
		}
		value = lipgloss.NewStyle().Foreground(ui.ColorText).Render(display)
	}

	return indicator + label + " " + value
}

func (m *Model) Breadcrumb() string {
	return m.opName
}

func (m *Model) KeyHints() []ui.KeyHint {
	if m.cursor >= 0 && m.cursor < len(m.fields) && m.fields[m.cursor].Type == FieldText {
		return []ui.KeyHint{
			{Key: "↑↓", Desc: "Field"},
			{Key: "Type", Desc: "Edit"},
			{Key: "←→", Desc: "Cursor"},
			{Key: "Bksp/Del", Desc: "Delete"},
			{Key: "Enter", Desc: "Execute"},
			{Key: "Esc", Desc: "Back"},
		}
	}
	return []ui.KeyHint{
		{Key: "↑↓", Desc: "Field"},
		{Key: "←→", Desc: "Change"},
		{Key: "Enter", Desc: "Execute"},
		{Key: "c", Desc: "Copy cmd"},
		{Key: "Esc", Desc: "Back"},
	}
}

func (m *Model) outputPath() string {
	ext := m.outputExtension()
	base := strings.TrimSuffix(filepath.Base(m.filePath), filepath.Ext(m.filePath))
	return filepath.Join(m.outputDir, base+"_"+operationSlug(m.opName)+"."+ext)
}

func (m *Model) outputExtension() string {
	switch m.opID {
	case operations.OpConvert:
		for _, f := range m.fields {
			if f.Label == "Format" {
				return f.Value
			}
		}
		return "mp4"
	case operations.OpExtractAudio:
		for _, f := range m.fields {
			if f.Label == "Format" {
				return f.Value
			}
		}
		return "mp3"
	case operations.OpMerge:
		ext := filepath.Ext(m.filePath)
		if ext != "" {
			return ext[1:]
		}
		return "mp4"
	case operations.OpGIF:
		return "gif"
	case operations.OpThumbnails:
		return "png"
	default:
		ext := filepath.Ext(m.filePath)
		if ext != "" {
			return ext[1:]
		}
		return "mp4"
	}
}

func (m *Model) buildFields() []Field {
	switch m.opID {
	case operations.OpConvert:
		return m.convertFields()
	case operations.OpExtractAudio:
		return m.extractAudioFields()
	case operations.OpResize:
		return m.resizeFields()
	case operations.OpTrim:
		return m.trimFields()
	case operations.OpCompress:
		return m.compressFields()
	case operations.OpMerge:
		return m.mergeFields()
	case operations.OpSubtitles:
		return m.subtitlesFields()
	case operations.OpWatermark:
		return m.watermarkFields()
	case operations.OpGIF:
		return m.gifFields()
	case operations.OpThumbnails:
		return m.thumbnailFields()
	case operations.OpAudio:
		return m.audioFields()
	case operations.OpFilters:
		return m.filtersFields()
	default:
		return m.convertFields()
	}
}

func (m *Model) convertFields() []Field {
	return []Field{
		{
			Label:    "Format",
			Type:     FieldSelect,
			Options:  []Option{{Label: "MP4", Value: "mp4"}, {Label: "MKV", Value: "mkv"}, {Label: "WebM", Value: "webm"}, {Label: "AVI", Value: "avi"}, {Label: "MOV", Value: "mov"}},
			Value:    "mp4",
			Selected: 0,
		},
		{
			Label:    "Codec",
			Type:     FieldSelect,
			Options:  []Option{{Label: "H.264", Value: "libx264"}, {Label: "H.265", Value: "libx265"}, {Label: "AV1", Value: "libsvtav1"}, {Label: "VP9", Value: "libvpx-vp9"}},
			Value:    "libx264",
			Selected: 0,
		},
		{
			Label:    "Quality",
			Type:     FieldSelect,
			Options:  []Option{{Label: "High (CRF 18)", Value: "18"}, {Label: "Balanced (CRF 23)", Value: "23"}, {Label: "Small (CRF 28)", Value: "28"}, {Label: "Tiny (CRF 32)", Value: "32"}},
			Value:    "23",
			Selected: 1,
		},
		{
			Label:    "Preset",
			Type:     FieldSelect,
			Options:  []Option{{Label: "Slow", Value: "slow"}, {Label: "Medium", Value: "medium"}, {Label: "Fast", Value: "fast"}, {Label: "Ultrafast", Value: "ultrafast"}},
			Value:    "medium",
			Selected: 1,
		},
		{
			Label:    "Audio",
			Type:     FieldSelect,
			Options:  []Option{{Label: "Copy", Value: "copy"}, {Label: "AAC", Value: "aac"}, {Label: "MP3", Value: "libmp3lame"}, {Label: "Opus", Value: "libopus"}},
			Value:    "copy",
			Selected: 0,
		},
	}
}

func (m *Model) extractAudioFields() []Field {
	return []Field{
		{
			Label:    "Format",
			Type:     FieldSelect,
			Options:  []Option{{Label: "MP3", Value: "mp3"}, {Label: "AAC", Value: "m4a"}, {Label: "FLAC", Value: "flac"}, {Label: "WAV", Value: "wav"}, {Label: "OGG", Value: "ogg"}, {Label: "Opus", Value: "opus"}},
			Value:    "mp3",
			Selected: 0,
		},
		{
			Label:    "Bitrate",
			Type:     FieldSelect,
			Options:  []Option{{Label: "320k (CD)", Value: "320k"}, {Label: "256k (High)", Value: "256k"}, {Label: "192k (Good)", Value: "192k"}, {Label: "128k (Podcast)", Value: "128k"}, {Label: "64k (Lo-fi)", Value: "64k"}},
			Value:    "192k",
			Selected: 2,
		},
	}
}

func (m *Model) resizeFields() []Field {
	return []Field{
		{
			Label:    "Resolution",
			Type:     FieldSelect,
			Options:  []Option{{Label: "4K (2160p)", Value: "2160"}, {Label: "1080p", Value: "1080"}, {Label: "720p", Value: "720"}, {Label: "480p", Value: "480"}, {Label: "360p", Value: "360"}},
			Value:    "1080",
			Selected: 1,
		},
		{
			Label:    "Aspect Ratio",
			Type:     FieldSelect,
			Options:  []Option{{Label: "Keep Original", Value: "keep"}, {Label: "16:9", Value: "16:9"}, {Label: "4:3", Value: "4:3"}, {Label: "Crop to Fit", Value: "crop"}},
			Value:    "keep",
			Selected: 0,
		},
		{
			Label:    "Codec",
			Type:     FieldSelect,
			Options:  []Option{{Label: "H.264", Value: "libx264"}, {Label: "H.265", Value: "libx265"}},
			Value:    "libx264",
			Selected: 0,
		},
	}
}

func (m *Model) trimFields() []Field {
	dur := ""
	if m.probeResult != nil {
		dur = ffmpeg.FormatDuration(time.Duration(m.probeResult.Format.Duration * float64(time.Second)))
	}
	return []Field{
		{
			Label:  "Start Time",
			Type:   FieldText,
			Value:  "00:00:00",
			Cursor: len([]rune("00:00:00")),
		},
		{
			Label:  "End Time",
			Type:   FieldText,
			Value:  dur,
			Cursor: len([]rune(dur)),
		},
		{
			Label:   "Lossless Cut",
			Type:    FieldToggle,
			Enabled: true,
			Value:   "true",
		},
	}
}

func (m *Model) compressFields() []Field {
	return m.compressFieldsWith(false, "23", "2500k")
}

func (m *Model) compressFieldsWith(twoPass bool, quality, bitrate string) []Field {
	qualityField := Field{
		Label:    "Quality",
		Type:     FieldSelect,
		Options:  []Option{{Label: "Visually Lossless", Value: "18"}, {Label: "Good", Value: "23"}, {Label: "Noticeable", Value: "28"}, {Label: "Heavy", Value: "32"}},
		Value:    quality,
		Selected: 1,
	}
	for i, opt := range qualityField.Options {
		if opt.Value == quality {
			qualityField.Selected = i
			qualityField.Value = quality
		}
	}
	bitrateField := Field{
		Label:  "Target Bitrate",
		Type:   FieldText,
		Value:  bitrate,
		Cursor: len([]rune(bitrate)),
	}

	first := qualityField
	if twoPass {
		first = bitrateField
	}

	return []Field{
		first,
		{
			Label:    "Codec",
			Type:     FieldSelect,
			Options:  []Option{{Label: "H.264 (Compatible)", Value: "libx264"}, {Label: "H.265 (Smaller)", Value: "libx265"}, {Label: "AV1 (Smallest)", Value: "libsvtav1"}},
			Value:    "libx264",
			Selected: 0,
		},
		{
			Label:    "Preset",
			Type:     FieldSelect,
			Options:  []Option{{Label: "Slow (Better)", Value: "slow"}, {Label: "Medium", Value: "medium"}, {Label: "Fast", Value: "fast"}},
			Value:    "medium",
			Selected: 1,
		},
		{
			Label:   "Two-Pass",
			Type:    FieldToggle,
			Enabled: twoPass,
			Value:   fmt.Sprintf("%t", twoPass),
		},
	}
}

func (m *Model) mergeFields() []Field {
	return []Field{
		{
			Label:    "Merge Mode",
			Type:     FieldSelect,
			Options:  []Option{{Label: "Concat (Copy Streams)", Value: "copy"}, {Label: "Concat (Re-encode)", Value: "reencode"}},
			Value:    "copy",
			Selected: 0,
		},
	}
}

func (m *Model) subtitlesFields() []Field {
	streams := []Option{{Label: "Track 1", Value: "0"}}
	if m.probeResult != nil {
		subs := m.probeResult.SubtitleStreams()
		if len(subs) > 0 {
			streams = nil
			for i, s := range subs {
				label := fmt.Sprintf("Track %d (%s)", i+1, strings.ToUpper(s.CodecName))
				if lang, ok := s.Tags["language"]; ok && lang != "" {
					label += " " + strings.ToUpper(lang)
				}
				streams = append(streams, Option{Label: label, Value: fmt.Sprintf("%d", i)})
			}
		}
	}

	return []Field{
		{
			Label:    "Subtitle Mode",
			Type:     FieldSelect,
			Options:  []Option{{Label: "Burn-in", Value: "burn"}, {Label: "Embed", Value: "embed"}},
			Value:    "burn",
			Selected: 0,
		},
		{
			Label:    "Subtitle Track",
			Type:     FieldSelect,
			Options:  streams,
			Value:    streams[0].Value,
			Selected: 0,
		},
	}
}

func (m *Model) watermarkFields() []Field {
	return []Field{
		{
			Label:    "Position",
			Type:     FieldSelect,
			Options:  []Option{{Label: "Top Left", Value: "top-left"}, {Label: "Top Right", Value: "top-right"}, {Label: "Bottom Left", Value: "bottom-left"}, {Label: "Bottom Right", Value: "bottom-right"}, {Label: "Center", Value: "center"}},
			Value:    "bottom-right",
			Selected: 3,
		},
		{
			Label:    "Opacity",
			Type:     FieldSelect,
			Options:  []Option{{Label: "25%", Value: "0.25"}, {Label: "50%", Value: "0.50"}, {Label: "75%", Value: "0.75"}},
			Value:    "0.50",
			Selected: 1,
		},
		{
			Label:    "Size",
			Type:     FieldSelect,
			Options:  []Option{{Label: "Small", Value: "160x60"}, {Label: "Medium", Value: "240x90"}, {Label: "Large", Value: "320x120"}},
			Value:    "240x90",
			Selected: 1,
		},
	}
}
func (m *Model) gifFields() []Field {
	return []Field{
		{
			Label:    "FPS",
			Type:     FieldSelect,
			Options:  []Option{{Label: "24 fps", Value: "24"}, {Label: "15 fps", Value: "15"}, {Label: "10 fps", Value: "10"}},
			Value:    "15",
			Selected: 1,
		},
		{
			Label:    "Width",
			Type:     FieldSelect,
			Options:  []Option{{Label: "640px", Value: "640"}, {Label: "480px", Value: "480"}, {Label: "320px", Value: "320"}},
			Value:    "480",
			Selected: 1,
		},
		{
			Label:  "Start Time",
			Type:   FieldText,
			Value:  "00:00:00",
			Cursor: len([]rune("00:00:00")),
		},
		{
			Label:  "Duration",
			Type:   FieldText,
			Value:  "5",
			Cursor: len([]rune("5")),
		},
	}
}

func (m *Model) thumbnailFields() []Field {
	return []Field{
		{
			Label:    "Mode",
			Type:     FieldSelect,
			Options:  []Option{{Label: "Single Frame", Value: "single"}, {Label: "Grid (4x4)", Value: "grid"}, {Label: "Every N Seconds", Value: "interval"}},
			Value:    "single",
			Selected: 0,
		},
		{
			Label:  "Timestamp",
			Type:   FieldText,
			Value:  "00:00:05",
			Cursor: len([]rune("00:00:05")),
		},
	}
}

func (m *Model) audioFields() []Field {
	return []Field{
		{
			Label:    "Operation",
			Type:     FieldSelect,
			Options:  []Option{{Label: "Normalize", Value: "normalize"}, {Label: "Volume Up", Value: "up"}, {Label: "Volume Down", Value: "down"}, {Label: "Fade In/Out", Value: "fade"}, {Label: "Remove Audio", Value: "remove"}},
			Value:    "normalize",
			Selected: 0,
		},
		{
			Label:    "Volume (dB)",
			Type:     FieldSelect,
			Options:  []Option{{Label: "+3 dB", Value: "3"}, {Label: "+6 dB", Value: "6"}, {Label: "-3 dB", Value: "-3"}, {Label: "-6 dB", Value: "-6"}},
			Value:    "3",
			Selected: 0,
		},
	}
}

func (m *Model) filtersFields() []Field {
	return []Field{
		{
			Label:    "Filter",
			Type:     FieldSelect,
			Options:  []Option{{Label: "Stabilize", Value: "vidstab"}, {Label: "Deinterlace", Value: "yadif"}, {Label: "Speed 2x", Value: "speed2"}, {Label: "Speed 0.5x", Value: "speed05"}, {Label: "Rotate 90", Value: "rotate90"}, {Label: "Flip Horizontal", Value: "hflip"}, {Label: "Flip Vertical", Value: "vflip"}},
			Value:    "vidstab",
			Selected: 0,
		},
	}
}

func (m *Model) buildCommand() *ffmpeg.Command {
	output := m.outputPath()
	cmd := ffmpeg.NewCommand(m.ffmpegPath, m.filePath, output)

	switch m.opID {
	case operations.OpMerge:
		m.buildMergeCommand(cmd)
	case operations.OpSubtitles:
		m.buildSubtitlesCommand(cmd)
	case operations.OpWatermark:
		m.buildWatermarkCommand(cmd)
	case operations.OpConvert:
		m.buildConvertCommand(cmd)
	case operations.OpExtractAudio:
		m.buildExtractAudioCommand(cmd)
	case operations.OpResize:
		m.buildResizeCommand(cmd)
	case operations.OpTrim:
		m.buildTrimCommand(cmd)
	case operations.OpCompress:
		m.buildCompressCommand(cmd)
	case operations.OpGIF:
		m.buildGIFCommand(cmd)
	case operations.OpThumbnails:
		m.buildThumbnailCommand(cmd)
	case operations.OpAudio:
		m.buildAudioCommand(cmd)
	case operations.OpFilters:
		m.buildFiltersCommand(cmd)
	}

	return cmd
}

func (m *Model) buildCommands() []*ffmpeg.Command {
	if m.opID == operations.OpFilters && m.fieldValue("Filter") == "vidstab" {
		if !m.vidstabSupported() {
			return []*ffmpeg.Command{m.buildDeshakeFallbackCommand()}
		}
		return m.buildStabilizeCommands()
	}
	if m.opID == operations.OpCompress && m.fieldEnabled("Two-Pass") {
		codec := m.fieldValue("Codec")
		if ffmpeg.TwoPassSupported(codec) {
			return ffmpeg.BuildTwoPassCommands(
				m.ffmpegPath, m.filePath, m.outputPath(),
				codec, m.fieldValue("Preset"),
				strings.TrimSpace(m.fieldValue("Target Bitrate")),
				ffmpeg.TwoPassStatsPrefix(),
			)
		}
	}
	return []*ffmpeg.Command{m.buildCommand()}
}

func (m *Model) buildStabilizeCommands() []*ffmpeg.Command {
	transformPath := filepath.Join(m.outputDir, ".nano-ffmpeg-vidstab.trf")
	escapedTransformPath := escapeSubtitlesPath(transformPath)

	detect := ffmpeg.NewCommand(m.ffmpegPath, m.filePath, "-")
	detect.AddVideoFilter(fmt.Sprintf("vidstabdetect=result='%s'", escapedTransformPath))
	detect.NoAudio()
	detect.AddArgs("-f", "null")

	transform := ffmpeg.NewCommand(m.ffmpegPath, m.filePath, m.outputPath())
	transform.AddVideoFilter(fmt.Sprintf("vidstabtransform=input='%s'", escapedTransformPath))
	transform.SetAudioCodec("copy")

	return []*ffmpeg.Command{detect, transform}
}

func (m *Model) buildDeshakeFallbackCommand() *ffmpeg.Command {
	cmd := ffmpeg.NewCommand(m.ffmpegPath, m.filePath, m.outputPath())
	cmd.AddVideoFilter("deshake")
	cmd.SetAudioCodec("copy")
	return cmd
}

func (m *Model) vidstabSupported() bool {
	if m.vidstabKnown {
		return m.vidstabOK
	}
	m.vidstabKnown = true
	m.vidstabOK = hasFFmpegFilter(m.ffmpegPath, "vidstabdetect") && hasFFmpegFilter(m.ffmpegPath, "vidstabtransform")
	return m.vidstabOK
}

func (m *Model) commandPreview() string {
	commands := m.buildCommands()
	if len(commands) == 0 {
		return ""
	}
	parts := make([]string, 0, len(commands))
	for _, cmd := range commands {
		parts = append(parts, cmd.String())
	}
	return strings.Join(parts, " && ")
}

func (m *Model) fallbackNotice() string {
	if m.opID == operations.OpFilters && m.fieldValue("Filter") == "vidstab" && !m.vidstabSupported() {
		return "vidstab filters unavailable in your ffmpeg build; using deshake fallback."
	}
	if m.opID == operations.OpCompress && !ffmpeg.TwoPassSupported(m.fieldValue("Codec")) {
		return "Two-pass not supported for this codec; toggle disabled."
	}
	return ""
}

func (m *Model) fieldValue(label string) string {
	for _, f := range m.fields {
		if f.Label == label {
			return f.Value
		}
	}
	return ""
}

func (m *Model) fieldEnabled(label string) bool {
	for _, f := range m.fields {
		if f.Label == label {
			return f.Enabled
		}
	}
	return false
}

func (m *Model) buildConvertCommand(cmd *ffmpeg.Command) {
	codec := m.fieldValue("Codec")
	cmd.SetVideoCodec(codec)
	crfVal := m.fieldValue("Quality")
	cmd.SetCRF(parseInt(crfVal))
	cmd.SetPresetForCodec(codec, m.fieldValue("Preset"))
	audio := m.fieldValue("Audio")
	cmd.SetAudioCodec(audio)
}

func (m *Model) buildExtractAudioCommand(cmd *ffmpeg.Command) {
	cmd.NoVideo()
	format := m.fieldValue("Format")
	switch format {
	case "mp3":
		cmd.SetAudioCodec("libmp3lame")
	case "m4a":
		cmd.SetAudioCodec("aac")
	case "flac":
		cmd.SetAudioCodec("flac")
	case "wav":
		cmd.SetAudioCodec("pcm_s16le")
	case "ogg":
		cmd.SetAudioCodec("libvorbis")
	case "opus":
		cmd.SetAudioCodec("libopus")
	}
	cmd.SetAudioBitrate(m.fieldValue("Bitrate"))
}

func (m *Model) buildResizeCommand(cmd *ffmpeg.Command) {
	height := parseInt(m.fieldValue("Resolution"))
	cmd.SetScaleHeight(height)
	cmd.SetVideoCodec(m.fieldValue("Codec"))
	cmd.SetAudioCodec("copy")
}

func (m *Model) buildTrimCommand(cmd *ffmpeg.Command) {
	if start := strings.TrimSpace(m.fieldValue("Start Time")); start != "" {
		cmd.SetStartTime(start)
	}
	if end := strings.TrimSpace(m.fieldValue("End Time")); end != "" {
		cmd.SetEndTime(end)
	}
	if m.fieldEnabled("Lossless Cut") {
		cmd.StreamCopy()
	}
}

func (m *Model) buildCompressCommand(cmd *ffmpeg.Command) {
	codec := m.fieldValue("Codec")
	cmd.SetVideoCodec(codec)
	cmd.SetCRF(parseInt(m.fieldValue("Quality")))
	cmd.SetPresetForCodec(codec, m.fieldValue("Preset"))
	cmd.SetAudioCodec("copy")
}
func (m *Model) buildMergeCommand(cmd *ffmpeg.Command) {
	listPath, err := m.writeMergeConcatFile()
	if err != nil {
		// Fallback to passthrough behavior if list generation fails.
		cmd.SetVideoCodec("copy")
		cmd.SetAudioCodec("copy")
		return
	}

	// ffconcat scripts auto-select the concat demuxer when they begin with:
	// "ffconcat version 1.0"
	cmd.Input = listPath

	if m.fieldValue("Merge Mode") == "reencode" {
		cmd.SetVideoCodec("libx264")
		cmd.SetAudioCodec("aac")
		cmd.SetPreset("medium")
		return
	}
	cmd.StreamCopy()
}

func (m *Model) buildSubtitlesCommand(cmd *ffmpeg.Command) {
	mode := m.fieldValue("Subtitle Mode")
	track := parseInt(m.fieldValue("Subtitle Track"))

	switch mode {
	case "embed":
		cmd.AddArgs("-map", "0")
		cmd.SetVideoCodec("copy")
		cmd.SetAudioCodec("copy")
		if m.outputExtension() == "mp4" {
			cmd.AddArgs("-c:s", "mov_text")
		} else {
			cmd.AddArgs("-c:s", "copy")
		}
	default: // burn-in
		if m.probeResult == nil || len(m.probeResult.SubtitleStreams()) == 0 {
			// No subtitle streams detected; avoid building an invalid filter.
			cmd.SetVideoCodec("copy")
			cmd.SetAudioCodec("copy")
			return
		}
		cmd.AddVideoFilter(fmt.Sprintf("subtitles='%s':si=%d", escapeSubtitlesPath(m.filePath), track))
		cmd.SetAudioCodec("copy")
	}
}

func (m *Model) buildWatermarkCommand(cmd *ffmpeg.Command) {
	opacity := m.fieldValue("Opacity")
	size := m.fieldValue("Size")
	position := m.fieldValue("Position")

	// Use a lavfi color source as the watermark layer and overlay it with position/opacity controls.
	cmd.AddArgs("-f", "lavfi")
	cmd.AddArgs("-i", fmt.Sprintf("color=c=white@%s:s=%s", opacity, size))
	cmd.AddArgs("-filter_complex", fmt.Sprintf("[0:v][1:v]overlay=%s[v]", overlayPosition(position)))
	cmd.AddArgs("-map", "[v]")
	cmd.AddArgs("-map", "0:a?")
	cmd.SetVideoCodec("libx264")
	cmd.SetAudioCodec("copy")
	cmd.SetPixelFormat("yuv420p")
}

func (m *Model) buildGIFCommand(cmd *ffmpeg.Command) {
	fps := m.fieldValue("FPS")
	width := m.fieldValue("Width")
	startTime := m.fieldValue("Start Time")
	duration := m.fieldValue("Duration")

	if startTime != "" && startTime != "00:00:00" {
		cmd.SetStartTime(startTime)
	}
	if duration != "" {
		cmd.SetDuration(duration)
	}
	filter := fmt.Sprintf("fps=%s,scale=%s:-1:flags=lanczos,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse", fps, width)
	cmd.AddArgs("-filter_complex", filter)
}

func (m *Model) buildThumbnailCommand(cmd *ffmpeg.Command) {
	mode := m.fieldValue("Mode")
	timestamp := m.fieldValue("Timestamp")

	switch mode {
	case "single":
		cmd.SetStartTime(timestamp)
		cmd.AddArgs("-frames:v", "1")
	case "grid":
		cmd.AddVideoFilter("select='not(mod(n\\,30))',scale=320:-1,tile=4x4")
		cmd.AddArgs("-frames:v", "1")
	case "interval":
		cmd.AddVideoFilter("fps=1/5")
	}
}

func (m *Model) buildAudioCommand(cmd *ffmpeg.Command) {
	op := m.fieldValue("Operation")
	const fadeDuration = 2.0
	switch op {
	case "normalize":
		cmd.AddAudioFilter("loudnorm")
		cmd.SetVideoCodec("copy")
	case "up":
		db := m.fieldValue("Volume (dB)")
		cmd.AddAudioFilter(fmt.Sprintf("volume=%sdB", db))
		cmd.SetVideoCodec("copy")
	case "down":
		db := m.fieldValue("Volume (dB)")
		cmd.AddAudioFilter(fmt.Sprintf("volume=%sdB", db))
		cmd.SetVideoCodec("copy")
	case "fade":
		duration := 0.0
		if m.probeResult != nil {
			duration = m.probeResult.Format.Duration
		}
		fadeOutStart := clampFadeOutStart(duration, fadeDuration)
		cmd.AddAudioFilter(fmt.Sprintf("afade=t=in:st=0:d=2,afade=t=out:st=%s:d=2", formatFFmpegSeconds(fadeOutStart)))
		cmd.SetVideoCodec("copy")
	case "remove":
		cmd.NoAudio()
		cmd.SetVideoCodec("copy")
	}
}

func clampFadeOutStart(duration float64, fadeDuration float64) float64 {
	if duration <= 0 || fadeDuration <= 0 || math.IsNaN(duration) || math.IsInf(duration, 0) {
		return 0
	}
	start := duration - fadeDuration
	if start < 0 {
		return 0
	}
	return start
}

func formatFFmpegSeconds(seconds float64) string {
	if seconds < 0 || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		return "0"
	}
	return strconv.FormatFloat(seconds, 'f', -1, 64)
}

func (m *Model) buildFiltersCommand(cmd *ffmpeg.Command) {
	filter := m.fieldValue("Filter")
	switch filter {
	case "vidstab":
		cmd.AddVideoFilter("vidstabdetect")
	case "yadif":
		cmd.AddVideoFilter("yadif")
		cmd.SetAudioCodec("copy")
	case "speed2":
		cmd.AddVideoFilter("setpts=0.5*PTS")
		cmd.AddAudioFilter("atempo=2.0")
	case "speed05":
		cmd.AddVideoFilter("setpts=2.0*PTS")
		cmd.AddAudioFilter("atempo=0.5")
	case "rotate90":
		cmd.AddVideoFilter("transpose=1")
		cmd.SetAudioCodec("copy")
	case "hflip":
		cmd.AddVideoFilter("hflip")
		cmd.SetAudioCodec("copy")
	case "vflip":
		cmd.AddVideoFilter("vflip")
		cmd.SetAudioCodec("copy")
	}
}

func parseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func textWithCursor(value string, cursor int) string {
	runes := []rune(value)
	cursor = clampCursor(cursor, len(runes))
	return string(runes[:cursor]) + "│" + string(runes[cursor:])
}

func operationSlug(name string) string {
	slug := strings.ToLower(name)
	replacer := strings.NewReplacer(" ", "_", "/", "_", "\\", "_", "-", "_")
	slug = replacer.Replace(slug)
	for strings.Contains(slug, "__") {
		slug = strings.ReplaceAll(slug, "__", "_")
	}
	return strings.Trim(slug, "_")
}

func overlayPosition(position string) string {
	switch position {
	case "top-left":
		return "20:20"
	case "top-right":
		return "W-w-20:20"
	case "bottom-left":
		return "20:H-h-20"
	case "center":
		return "(W-w)/2:(H-h)/2"
	default: // bottom-right
		return "W-w-20:H-h-20"
	}
}

func escapeSubtitlesPath(path string) string {
	escaped := strings.ReplaceAll(path, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, ":", "\\:")
	escaped = strings.ReplaceAll(escaped, "'", "\\'")
	return escaped
}

func hasFFmpegFilter(ffmpegPath string, filterName string) bool {
	out, err := exec.Command(ffmpegPath, "-hide_banner", "-filters").Output()
	if err != nil {
		return false
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[1] == filterName {
			return true
		}
	}
	return false
}

func (m *Model) writeMergeConcatFile() (string, error) {
	sourcePath := filepath.Clean(m.filePath)
	sourceDir := filepath.Dir(sourcePath)
	sourceExt := strings.ToLower(filepath.Ext(sourcePath))

	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return "", err
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.ToLower(filepath.Ext(e.Name())) != sourceExt {
			continue
		}
		files = append(files, e.Name())
	}
	if len(files) == 0 {
		files = []string{filepath.Base(sourcePath)}
	}
	sort.Strings(files)

	var b strings.Builder
	b.WriteString("ffconcat version 1.0\n")
	for _, f := range files {
		b.WriteString("file '")
		b.WriteString(strings.ReplaceAll(f, "'", "\\'"))
		b.WriteString("'\n")
	}

	listPath := filepath.Join(sourceDir, ".nano-ffmpeg-merge.ffconcat")
	if err := os.WriteFile(listPath, []byte(b.String()), 0644); err != nil {
		return "", err
	}
	return listPath, nil
}
