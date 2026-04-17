package result

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dgr8akki/nano-ffmpeg/internal/screens"
)

func TestResultNew_ReadsOutputSize(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "out.mp4")
	payload := bytes.Repeat([]byte("x"), 2048)
	if err := os.WriteFile(outPath, payload, 0o644); err != nil {
		t.Fatalf("write output: %v", err)
	}

	m := New(outPath, 4096)
	if m.outputSize != int64(len(payload)) {
		t.Fatalf("outputSize: got %d, want %d", m.outputSize, len(payload))
	}
	if m.inputSize != 4096 {
		t.Fatalf("inputSize: got %d, want 4096", m.inputSize)
	}
}

func TestResultNew_MissingFileLeavesSizeZero(t *testing.T) {
	m := New("/no/such/file.mp4", 1024)
	if m.outputSize != 0 {
		t.Fatalf("expected outputSize 0, got %d", m.outputSize)
	}
	// View should still render without panic.
	if m.View() == "" {
		t.Fatal("expected non-empty view")
	}
}

func TestResultView_RendersSuccess(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "clip.mkv")
	if err := os.WriteFile(outPath, []byte("data"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	m := New(outPath, 2000)
	view := m.View()
	for _, want := range []string{
		"Done",
		outPath,
		"Do another operation",
		"Quit",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q", want)
		}
	}
}

func TestRenderSizeComparison_Smaller(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "out.mp4")
	if err := os.WriteFile(outPath, bytes.Repeat([]byte("x"), 512), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m := New(outPath, 1024)
	got := m.renderSizeComparison()
	if !strings.Contains(got, "smaller") {
		t.Fatalf("expected 'smaller' label, got: %s", got)
	}
}

func TestRenderSizeComparison_Larger(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "out.mp4")
	if err := os.WriteFile(outPath, bytes.Repeat([]byte("x"), 2048), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m := New(outPath, 1024)
	got := m.renderSizeComparison()
	if !strings.Contains(got, "larger") {
		t.Fatalf("expected 'larger' label, got: %s", got)
	}
}

func TestRenderSizeComparison_EmptyWhenNoSizes(t *testing.T) {
	m := New("/no/such/file.mp4", 0)
	if got := m.renderSizeComparison(); got != "" {
		t.Fatalf("expected empty comparison, got %q", got)
	}
}

func TestResultUpdate_EnterOnDefaultOptionNavigatesHome(t *testing.T) {
	m := New("/tmp/out.mp4", 0)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected cmd for Enter")
	}
	nav, ok := cmd().(screens.NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T", cmd())
	}
	if nav.Screen != screens.ScreenHome {
		t.Fatalf("expected ScreenHome, got %d", nav.Screen)
	}
}

func TestResultUpdate_EnterOnQuitReturnsTeaQuit(t *testing.T) {
	m := New("/tmp/out.mp4", 0)
	for i := 0; i < len(m.options)-1; i++ {
		s, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = s.(*Model)
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected cmd from Enter on Quit option")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", cmd())
	}
}

func TestResultUpdate_EnterOnBuyMeACoffeeOpensURL(t *testing.T) {
	orig := openURL
	t.Cleanup(func() { openURL = orig })

	var opened string
	openURL = func(url string) error {
		opened = url
		return nil
	}

	m := New("/tmp/out.mp4", 0)
	s, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = s.(*Model)
	if m.options[m.cursor] != "Buy me a coffee ☕" {
		t.Fatalf("expected cursor on coffee option, got %q", m.options[m.cursor])
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected cmd from Enter on coffee option")
	}
	if msg := cmd(); msg != nil {
		t.Fatalf("expected nil msg, got %T", msg)
	}
	if opened != buyMeACoffeeURL {
		t.Fatalf("expected openURL called with %q, got %q", buyMeACoffeeURL, opened)
	}
}

func TestResultView_RendersCoffeeOption(t *testing.T) {
	m := New("/tmp/out.mp4", 0)
	if !strings.Contains(m.View(), "Buy me a coffee") {
		t.Fatal("expected view to render Buy me a coffee option")
	}
}

func TestResultUpdate_EscNavigatesHome(t *testing.T) {
	m := New("/tmp/out.mp4", 0)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected cmd")
	}
	nav, ok := cmd().(screens.NavigateMsg)
	if !ok || nav.Screen != screens.ScreenHome {
		t.Fatalf("expected NavigateMsg home, got %T %+v", cmd(), cmd())
	}
}

func TestResultUpdate_NavigationClamped(t *testing.T) {
	m := New("/tmp/out.mp4", 0)
	s, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = s.(*Model)
	if m.cursor != 0 {
		t.Fatalf("expected cursor 0 after up at top, got %d", m.cursor)
	}
	for i := 0; i < 5; i++ {
		s, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = s.(*Model)
	}
	if m.cursor != len(m.options)-1 {
		t.Fatalf("expected cursor %d, got %d", len(m.options)-1, m.cursor)
	}
}

func TestResultBreadcrumbAndKeyHints(t *testing.T) {
	m := New("/tmp/out.mp4", 0)
	if got := m.Breadcrumb(); got != "Result" {
		t.Fatalf("breadcrumb: got %q", got)
	}
	if len(m.KeyHints()) == 0 {
		t.Fatal("expected key hints")
	}
}

func TestResultUpdate_WindowSizeStored(t *testing.T) {
	m := New("/tmp/out.mp4", 0)
	s, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 25})
	m = s.(*Model)
	if m.width != 100 || m.height != 25 {
		t.Fatalf("WindowSizeMsg ignored: %dx%d", m.width, m.height)
	}
}

func TestResultInit_ReturnsNoCmd(t *testing.T) {
	if New("/tmp/out.mp4", 0).Init() != nil {
		t.Fatal("expected Init to return nil")
	}
}
