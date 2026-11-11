package settings

import (
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dgr8akki/nano-ffmpeg/internal/ffmpeg"
	"github.com/dgr8akki/nano-ffmpeg/internal/screens/operations"
)

func TestBuildFields_FirstP0OpsDoNotFallbackToConvert(t *testing.T) {
	subProbe := &ffmpeg.ProbeResult{
		Streams: []ffmpeg.ProbeStream{
			{CodecType: "subtitle", CodecName: "subrip"},
		},
	}

	tests := []struct {
		name       string
		opID       operations.OperationID
		opName     string
		probe      *ffmpeg.ProbeResult
		firstLabel string
	}{
		{
			name:       "merge fields",
			opID:       operations.OpMerge,
			opName:     "Merge / Concat",
			firstLabel: "Merge Mode",
		},
		{
			name:       "subtitles fields",
			opID:       operations.OpSubtitles,
			opName:     "Add Subtitles",
			probe:      subProbe,
			firstLabel: "Subtitle Mode",
		},
		{
			name:       "watermark fields",
			opID:       operations.OpWatermark,
			opName:     "Watermark",
			firstLabel: "Position",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.opID, tt.opName, "/tmp/input.mp4", tt.probe, "/usr/bin/ffmpeg")
			if len(m.fields) == 0 {
				t.Fatalf("expected fields for %s", tt.opName)
			}
			if m.fields[0].Label != tt.firstLabel {
				t.Fatalf("expected first field %q, got %q", tt.firstLabel, m.fields[0].Label)
			}
		})
	}
}

func hasArg(args []string, needle string) bool {
	for _, a := range args {
		if a == needle {
			return true
		}
	}
	return false
}

func TestBuildCommand_MergeUsesConcatScript(t *testing.T) {
	dir := t.TempDir()
	inputA := filepath.Join(dir, "clip_a.mp4")
	inputB := filepath.Join(dir, "clip_b.mp4")
	if err := os.WriteFile(inputA, []byte("a"), 0644); err != nil {
		t.Fatalf("write inputA: %v", err)
	}
	if err := os.WriteFile(inputB, []byte("b"), 0644); err != nil {
		t.Fatalf("write inputB: %v", err)
	}

	m := New(operations.OpMerge, "Merge / Concat", inputA, nil, "/usr/bin/ffmpeg")
	cmd := m.buildCommand()
	args := strings.Join(cmd.Build(), " ")

	if !strings.HasSuffix(cmd.Input, "nano-ffmpeg-merge.ffconcat") || filepath.Dir(cmd.Input) == dir {
		t.Fatalf("expected ffconcat input file, got %q", cmd.Input)
	}
	if !strings.Contains(args, "-c copy") {
		t.Fatalf("expected stream-copy merge command, got: %s", args)
	}
	if !strings.Contains(filepath.Base(cmd.Output), "merge_concat") {
		t.Fatalf("expected sanitized merge output name, got %q", cmd.Output)
	}

	script, err := os.ReadFile(cmd.Input)
	if err != nil {
		t.Fatalf("read ffconcat script: %v", err)
	}
	content := string(script)
	if !strings.Contains(content, "ffconcat version 1.0") {
		t.Fatalf("expected ffconcat header, got:\n%s", content)
	}
	if !strings.Contains(content, "file '"+inputA+"'") || !strings.Contains(content, "file '"+inputB+"'") {
		t.Fatalf("expected both media files in concat script, got:\n%s", content)
	}
}

func TestBuildCommand_SubtitlesBurnAndEmbed(t *testing.T) {
	probe := &ffmpeg.ProbeResult{
		Streams: []ffmpeg.ProbeStream{
			{CodecType: "video", CodecName: "h264"},
			{CodecType: "subtitle", CodecName: "subrip"},
		},
	}

	m := New(operations.OpSubtitles, "Add Subtitles", "/tmp/input.mp4", probe, "/usr/bin/ffmpeg")

	burnArgs := strings.Join(m.buildCommand().Build(), " ")
	if !strings.Contains(burnArgs, "subtitles=") {
		t.Fatalf("expected burn-in subtitles filter, got: %s", burnArgs)
	}

	if !setFieldSelectValue(m, "Subtitle Mode", "embed") {
		t.Fatalf("failed to set subtitle mode to embed")
	}
	embedArgs := strings.Join(m.buildCommand().Build(), " ")
	if !strings.Contains(embedArgs, "-map 0") {
		t.Fatalf("expected stream mapping in embed mode, got: %s", embedArgs)
	}
	if !strings.Contains(embedArgs, "-c:s mov_text") {
		t.Fatalf("expected mov_text subtitle codec for mp4 output, got: %s", embedArgs)
	}
}

func TestBuildCommand_WatermarkUsesOverlayFilter(t *testing.T) {
	m := New(operations.OpWatermark, "Watermark", "/tmp/input.mp4", nil, "/usr/bin/ffmpeg")
	args := strings.Join(m.buildCommand().Build(), " ")

	if !strings.Contains(args, "-f lavfi -i color=c=white@") {
		t.Fatalf("expected lavfi watermark source, got: %s", args)
	}
	if !strings.Contains(args, "overlay=") {
		t.Fatalf("expected overlay filter, got: %s", args)
	}
	if !strings.Contains(args, "-map [v]") || !strings.Contains(args, "-map 0:a?") {
		t.Fatalf("expected mapped filtered video + optional audio, got: %s", args)
	}
}

func TestBuildCommands_StabilizeUsesTwoPassPipeline(t *testing.T) {
	m := New(operations.OpFilters, "Video Filters", "/tmp/input.mp4", nil, "/usr/bin/ffmpeg")
	m.vidstabKnown = true
	m.vidstabOK = true
	commands := m.buildCommands()
	if len(commands) != 2 {
		t.Fatalf("expected two stabilize commands, got %d", len(commands))
	}

	detectArgs := strings.Join(commands[0].Build(), " ")
	if !strings.Contains(detectArgs, "vidstabdetect=result=") {
		t.Fatalf("expected vidstabdetect in first pass, got: %s", detectArgs)
	}
	if !strings.Contains(detectArgs, "-f null -") {
		t.Fatalf("expected null-output first pass, got: %s", detectArgs)
	}
	if !strings.Contains(detectArgs, "nano-ffmpeg-vidstab.trf") {
		t.Fatalf("expected shared transform file in first pass, got: %s", detectArgs)
	}

	transformArgs := strings.Join(commands[1].Build(), " ")
	if !strings.Contains(transformArgs, "vidstabtransform=input=") {
		t.Fatalf("expected vidstabtransform in second pass, got: %s", transformArgs)
	}
	if !strings.Contains(transformArgs, "-c:a copy") {
		t.Fatalf("expected audio copy in second pass, got: %s", transformArgs)
	}
	if !strings.Contains(transformArgs, "nano-ffmpeg-vidstab.trf") {
		t.Fatalf("expected shared transform file in second pass, got: %s", transformArgs)
	}
}

func TestBuildCommands_StabilizeFallsBackToDeshakeWhenVidstabUnavailable(t *testing.T) {
	m := New(operations.OpFilters, "Video Filters", "/tmp/input.mp4", nil, "/usr/bin/ffmpeg")
	m.vidstabKnown = true
	m.vidstabOK = false

	commands := m.buildCommands()
	if len(commands) != 1 {
		t.Fatalf("expected single fallback command, got %d", len(commands))
	}
	args := strings.Join(commands[0].Build(), " ")
	if !strings.Contains(args, "-vf deshake") {
		t.Fatalf("expected deshake fallback command, got: %s", args)
	}
	if strings.Contains(args, "vidstabdetect") || strings.Contains(args, "vidstabtransform") {
		t.Fatalf("did not expect vidstab filters in fallback command, got: %s", args)
	}
}

func TestFallbackNotice_ShownWhenVidstabUnavailable(t *testing.T) {
	m := New(operations.OpFilters, "Video Filters", "/tmp/input.mp4", nil, "/usr/bin/ffmpeg")
	m.vidstabKnown = true
	m.vidstabOK = false

	notice := m.fallbackNotice()
	if notice == "" {
		t.Fatal("expected fallback notice when vidstab is unavailable")
	}
	if !strings.Contains(notice, "deshake fallback") {
		t.Fatalf("expected deshake fallback message, got: %q", notice)
	}
}

func TestFallbackNotice_NotShownForNonStabilizeFilter(t *testing.T) {
	m := New(operations.OpFilters, "Video Filters", "/tmp/input.mp4", nil, "/usr/bin/ffmpeg")
	m.vidstabKnown = true
	m.vidstabOK = false
	if !setFieldSelectValue(m, "Filter", "yadif") {
		t.Fatal("failed to switch filter to yadif")
	}

	if notice := m.fallbackNotice(); notice != "" {
		t.Fatalf("did not expect fallback notice for non-stabilize filter, got: %q", notice)
	}
}

func TestBuildCommands_NonStabilizeFilterRemainsSinglePass(t *testing.T) {
	m := New(operations.OpFilters, "Video Filters", "/tmp/input.mp4", nil, "/usr/bin/ffmpeg")
	if !setFieldSelectValue(m, "Filter", "yadif") {
		t.Fatalf("failed to switch filter to yadif")
	}

	commands := m.buildCommands()
	if len(commands) != 1 {
		t.Fatalf("expected single command for non-stabilize filter, got %d", len(commands))
	}
	args := strings.Join(commands[0].Build(), " ")
	if strings.Contains(args, "vidstabdetect") || strings.Contains(args, "vidstabtransform") {
		t.Fatalf("did not expect stabilize filters for yadif command, got: %s", args)
	}
}

func TestBuildCommand_AudioFadeUsesDurationBasedStart(t *testing.T) {
	probe := &ffmpeg.ProbeResult{
		Format: ffmpeg.ProbeFormat{Duration: 12.5},
	}
	m := New(operations.OpAudio, "Audio", "/tmp/input.mp4", probe, "/usr/bin/ffmpeg")
	if !setFieldSelectValue(m, "Operation", "fade") {
		t.Fatal("failed to set audio operation to fade")
	}

	args := strings.Join(m.buildCommand().Build(), " ")
	if !strings.Contains(args, "-af afade=t=in:st=0:d=2,afade=t=out:st=10.5:d=2") {
		t.Fatalf("expected duration-based fade-out start time, got: %s", args)
	}
}

func TestBuildCommand_AudioFadeClampsStartForShortOrUnknownDuration(t *testing.T) {
	tests := []struct {
		name  string
		probe *ffmpeg.ProbeResult
	}{
		{
			name:  "unknown duration",
			probe: nil,
		},
		{
			name: "short duration",
			probe: &ffmpeg.ProbeResult{
				Format: ffmpeg.ProbeFormat{Duration: 1.2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(operations.OpAudio, "Audio", "/tmp/input.mp4", tt.probe, "/usr/bin/ffmpeg")
			if !setFieldSelectValue(m, "Operation", "fade") {
				t.Fatal("failed to set audio operation to fade")
			}

			args := strings.Join(m.buildCommand().Build(), " ")
			if !strings.Contains(args, "-af afade=t=in:st=0:d=2,afade=t=out:st=0:d=2") {
				t.Fatalf("expected clamped fade-out start time, got: %s", args)
			}
		})
	}
}
func TestTrimFields_DefaultEndTimeUsesFFmpegTimestamp(t *testing.T) {
	probe := &ffmpeg.ProbeResult{
		Format: ffmpeg.ProbeFormat{Duration: 65},
	}

	m := New(operations.OpTrim, "Trim", "/tmp/input.mp4", probe, "/usr/bin/ffmpeg")
	if got := m.fieldValue("End Time"); got != "00:01:05" {
		t.Fatalf("expected ffmpeg-compatible end time 00:01:05, got %q", got)
	}
}

func TestTrimTextFieldsAreEditableFromKeyboardInput(t *testing.T) {
	m := New(operations.OpTrim, "Trim", "/tmp/input.mp4", nil, "/usr/bin/ffmpeg")

	if got := m.fieldValue("Start Time"); got != "00:00:00" {
		t.Fatalf("expected default start time, got %q", got)
	}

	sendKey(m, tea.KeyMsg{Type: tea.KeyBackspace})
	sendKey(m, tea.KeyMsg{Type: tea.KeyBackspace})
	sendKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("15")})

	if got := m.fieldValue("Start Time"); got != "00:00:15" {
		t.Fatalf("expected edited start time 00:00:15, got %q", got)
	}

	sendKey(m, tea.KeyMsg{Type: tea.KeyLeft})
	sendKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("9")})
	if got := m.fieldValue("Start Time"); got != "00:00:195" {
		t.Fatalf("expected left-move insertion in text field, got %q", got)
	}
	if got := m.fields[0].Cursor; got != len([]rune("00:00:195"))-1 {
		t.Fatalf("expected cursor position to track text edits, got %d", got)
	}
}

func TestBuildCommand_TrimUsesEditedTextFieldValues(t *testing.T) {
	m := New(operations.OpTrim, "Trim", "/tmp/input.mp4", nil, "/usr/bin/ffmpeg")

	sendKey(m, tea.KeyMsg{Type: tea.KeyBackspace})
	sendKey(m, tea.KeyMsg{Type: tea.KeyBackspace})
	sendKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("10")})

	sendKey(m, tea.KeyMsg{Type: tea.KeyDown})
	sendKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("00:00:45")})

	args := strings.Join(m.buildCommand().Build(), " ")
	if !strings.Contains(args, "-ss 00:00:10") {
		t.Fatalf("expected edited start time in trim command, got: %s", args)
	}
	if !strings.Contains(args, "-to 00:00:45") {
		t.Fatalf("expected edited end time in trim command, got: %s", args)
	}
}

func TestBuildCommand_TrimSkipsEmptyEndTime(t *testing.T) {
	m := New(operations.OpTrim, "Trim", "/tmp/input.mp4", nil, "/usr/bin/ffmpeg")
	args := m.buildCommand().Build()
	if hasArg(args, "-to") {
		t.Fatalf("did not expect -to when end time is empty, got args: %v", args)
	}
}

func sendKey(m *Model, key tea.KeyMsg) {
	_, _ = m.Update(key)
}

func setFieldSelectValue(m *Model, label string, value string) bool {
	for i := range m.fields {
		if m.fields[i].Label != label || m.fields[i].Type != FieldSelect {
			continue
		}
		for optionIdx, opt := range m.fields[i].Options {
			if opt.Value == value {
				m.fields[i].Selected = optionIdx
				m.fields[i].Value = value
				return true
			}
		}
	}
	return false
}

func TestParseTime(t *testing.T) {
	ok := map[string]float64{"5": 5, "1.5": 1.5, "0:03": 3, "01:30": 90, "00:00:07.25": 7.25, "1:00:00": 3600}
	for in, want := range ok {
		if got, valid := parseTime(in); !valid || got != want {
			t.Errorf("parseTime(%q) = %v,%v want %v", in, got, valid, want)
		}
	}
	for _, in := range []string{"", "abc", "1:60", "-1", "1:2:3:4", "1.5:00", "NaN", "Inf"} {
		if _, valid := parseTime(in); valid {
			t.Errorf("parseTime(%q) should be invalid", in)
		}
	}
}

func TestEscapeFilterPath(t *testing.T) {
	cases := map[string]string{
		"/a/b.mkv":            "/a/b.mkv",
		`C:\Users\x.mkv`:      `C\\:\\\\Users\\\\x.mkv`,
		"/v/it's [a],b;c.mkv": `/v/it\\\'s \[a\]\,b\;c.mkv`,
	}
	for in, want := range cases {
		if got := escapeFilterPath(in); got != want {
			t.Errorf("escapeFilterPath(%q) = %q want %q", in, got, want)
		}
	}
}

func TestMergeExcludesItsOwnOutput(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.mp4")
	for _, n := range []string{"a.mp4", "b.mp4", "a_merge_concat.mp4"} {
		os.WriteFile(filepath.Join(dir, n), []byte("x"), 0644)
	}
	m := New(operations.OpMerge, "Merge / Concat", a, nil, "/usr/bin/ffmpeg")
	script, _ := os.ReadFile(m.buildCommand().Input)
	if strings.Contains(string(script), "a_merge_concat.mp4") {
		t.Fatalf("merge list includes its own output:\n%s", script)
	}
}
