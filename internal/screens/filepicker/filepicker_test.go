package filepicker

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dgr8akki/nano-ffmpeg/internal/screens"
)

func TestIsMediaFile(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"clip.mp4", true},
		{"song.MP3", true},
		{"SAMPLE.MKV", true},
		{"script.go", false},
		{"notes.txt", false},
		{"no-extension", false},
		{"archive.tar.gz", false},
	}
	for _, tt := range tests {
		if got := isMediaFile(tt.name); got != tt.want {
			t.Errorf("isMediaFile(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestFilepickerFormatSize(t *testing.T) {
	cases := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{int64(1.5 * 1024 * 1024), "1.5 MB"},
	}
	for _, c := range cases {
		if got := formatSize(c.bytes); got != c.want {
			t.Errorf("formatSize(%d) = %q, want %q", c.bytes, got, c.want)
		}
	}
}

func TestFilepickerParseFPS(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"30/1", 30},
		{"0/0", 0},
		{"bad", 0},
		{"", 0},
	}
	for _, c := range cases {
		got := parseFPS(c.in)
		if math.Abs(got-c.want) > 0.001 {
			t.Errorf("parseFPS(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestFilepickerNew_UsesProvidedStartDir(t *testing.T) {
	dir := t.TempDir()
	m := New("ffprobe", dir)
	if m.currentDir != dir {
		t.Fatalf("currentDir: got %q, want %q", m.currentDir, dir)
	}
}

func TestFilepickerNew_DefaultsToHomeWhenStartDirBlank(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	m := New("ffprobe", "")
	if m.currentDir != home {
		t.Fatalf("currentDir: got %q, want %q", m.currentDir, home)
	}
}

func TestFilepickerLoadDir_ListsDirectoriesBeforeFiles(t *testing.T) {
	dir := t.TempDir()

	// dotfile (should be hidden), regular file, two subdirs
	for _, name := range []string{".hidden", "video.mp4", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	for _, name := range []string{"zdir", "adir"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	m := New("ffprobe", dir)

	if len(m.entries) != 4 {
		t.Fatalf("expected 4 entries (2 dirs + 2 files, dotfile hidden), got %d: %+v", len(m.entries), m.entries)
	}

	// Directories first (preserved in readdir order for this test).
	if !m.entries[0].isDir || !m.entries[1].isDir {
		t.Fatalf("expected directories first, got %+v", m.entries[:2])
	}
	// Files next
	if m.entries[2].isDir || m.entries[3].isDir {
		t.Fatalf("expected files last, got %+v", m.entries[2:])
	}
	// Sizes populated for files
	for _, e := range m.entries[2:] {
		if e.size != 1 {
			t.Errorf("expected size 1 for file %q, got %d", e.name, e.size)
		}
	}
}

func TestFilepickerUpdate_NavigatesEntriesWithClamping(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 3; i++ {
		if err := os.WriteFile(filepath.Join(dir, "f"+string(rune('a'+i))+".mp4"), []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	m := New("ffprobe", dir)
	// up from 0 stays 0
	s, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = s.(*Model)
	if m.cursor != 0 {
		t.Fatalf("up at top should stay 0, got %d", m.cursor)
	}

	for i := 0; i < 10; i++ {
		s, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = s.(*Model)
	}
	if m.cursor >= len(m.entries) {
		t.Fatalf("cursor escaped bounds: %d >= %d", m.cursor, len(m.entries))
	}
}

func TestFilepickerUpdate_BackspaceGoesToParent(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	m := New("ffprobe", child)
	s, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = s.(*Model)

	if m.currentDir != parent {
		t.Fatalf("expected parent %q, got %q", parent, m.currentDir)
	}
}

func TestFilepickerUpdate_SlashEntersPathInputMode(t *testing.T) {
	dir := t.TempDir()
	m := New("ffprobe", dir)

	s, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = s.(*Model)
	if !m.pathInput {
		t.Fatal("expected pathInput mode enabled")
	}
	if !strings.HasPrefix(m.pathText, dir) {
		t.Fatalf("expected pathText seeded with currentDir, got %q", m.pathText)
	}
}

func TestFilepickerUpdatePathInput_EnterNavigatesToDir(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	m := New("ffprobe", root)
	m.pathInput = true
	m.pathText = target

	s, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = s.(*Model)

	if m.pathInput {
		t.Fatal("expected pathInput mode cleared after enter")
	}
	if m.currentDir != target {
		t.Fatalf("expected currentDir %q, got %q", target, m.currentDir)
	}
}

func TestFilepickerUpdatePathInput_EnterOnMissingPathSetsErr(t *testing.T) {
	dir := t.TempDir()
	m := New("ffprobe", dir)
	m.pathInput = true
	m.pathText = filepath.Join(dir, "does-not-exist")

	s, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = s.(*Model)
	if m.err == nil {
		t.Fatal("expected err for missing path")
	}
}

func TestFilepickerUpdatePathInput_EscCancels(t *testing.T) {
	m := New("ffprobe", t.TempDir())
	m.pathInput = true
	m.pathText = "/tmp/foo"

	s, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = s.(*Model)

	if m.pathInput {
		t.Fatal("expected pathInput cleared by esc")
	}
}

func TestFilepickerUpdatePathInput_BackspaceEditsText(t *testing.T) {
	m := New("ffprobe", t.TempDir())
	m.pathInput = true
	m.pathText = "/tmp/abc"

	s, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = s.(*Model)
	if m.pathText != "/tmp/ab" {
		t.Fatalf("expected /tmp/ab, got %q", m.pathText)
	}
}

func TestFilepickerUpdateBrowser_EscEmitsBack(t *testing.T) {
	m := New("ffprobe", t.TempDir())
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected cmd from esc")
	}
	if _, ok := cmd().(screens.BackMsg); !ok {
		t.Fatalf("expected BackMsg, got %T", cmd())
	}
}

func TestFilepickerKeyHints_ReflectMode(t *testing.T) {
	m := New("ffprobe", t.TempDir())

	browser := m.KeyHints()
	var keys []string
	for _, h := range browser {
		keys = append(keys, h.Key)
	}
	joined := strings.Join(keys, ",")
	for _, want := range []string{"↑↓", "Enter", "Bksp", "/", "Esc"} {
		if !strings.Contains(joined, want) {
			t.Errorf("browser hints missing %q: %s", want, joined)
		}
	}

	m.pathInput = true
	pathMode := m.KeyHints()
	if len(pathMode) == 2 && pathMode[0].Key == "Enter" && pathMode[1].Key == "Esc" {
		return
	}
	t.Fatalf("unexpected path-input hints: %+v", pathMode)
}

func TestFilepickerView_RendersCurrentDirectory(t *testing.T) {
	dir := t.TempDir()
	m := New("ffprobe", dir)
	view := m.View()
	if !strings.Contains(view, dir) {
		t.Fatalf("view missing current dir %q:\n%s", dir, view)
	}
}

func TestFilepickerBreadcrumb(t *testing.T) {
	m := New("ffprobe", t.TempDir())
	if got := m.Breadcrumb(); got != "File Picker" {
		t.Fatalf("breadcrumb: got %q", got)
	}
}

func TestFilepickerInit_ReturnsNoCmd(t *testing.T) {
	m := New("ffprobe", t.TempDir())
	if m.Init() != nil {
		t.Fatal("expected Init to return nil")
	}
}

func TestFilepickerUpdate_WindowSizeStored(t *testing.T) {
	m := New("ffprobe", t.TempDir())
	s, _ := m.Update(tea.WindowSizeMsg{Width: 70, Height: 22})
	m = s.(*Model)
	if m.width != 70 || m.height != 22 {
		t.Fatalf("WindowSizeMsg ignored: %dx%d", m.width, m.height)
	}
}

func TestFilepickerUpdate_EnterOnDirectoryLoadsContents(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "nested")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(child, "clip.mp4"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	m := New("ffprobe", parent)
	// cursor is at first entry (should be the child dir)
	if len(m.entries) == 0 || !m.entries[0].isDir {
		t.Fatalf("expected at least one dir entry first, got %+v", m.entries)
	}

	s, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = s.(*Model)

	if m.currentDir != child {
		t.Fatalf("expected descend into %q, got %q", child, m.currentDir)
	}
}

func TestFilepickerVisibleLines_FloorEnforced(t *testing.T) {
	m := &Model{height: 0}
	if got := m.visibleLines(); got < 5 {
		t.Fatalf("expected visible lines >= 5, got %d", got)
	}
}

func TestFilepickerLoadDir_FollowsDirectorySymlinks(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "real")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "clip.mp4"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write inside target: %v", err)
	}

	browse := t.TempDir()
	link := filepath.Join(browse, "videos")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	m := New("ffprobe", browse)
	if len(m.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d: %+v", len(m.entries), m.entries)
	}
	if !m.entries[0].isDir {
		t.Fatalf("symlinked directory should be marked isDir: %+v", m.entries[0])
	}

	s, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = s.(*Model)
	if m.currentDir != link {
		t.Fatalf("expected to enter %q, got %q", link, m.currentDir)
	}
	if len(m.entries) != 1 || m.entries[0].name != "clip.mp4" {
		t.Fatalf("expected to list target contents, got %+v", m.entries)
	}
}

func TestFilepickerLoadDir_SkipsBrokenSymlinks(t *testing.T) {
	dir := t.TempDir()
	if err := os.Symlink(filepath.Join(dir, "missing-target"), filepath.Join(dir, "broken")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ok.mp4"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	m := New("ffprobe", dir)
	for _, e := range m.entries {
		if e.name == "broken" {
			t.Fatalf("broken symlink should be skipped, got entries: %+v", m.entries)
		}
	}
	if len(m.entries) != 1 || m.entries[0].name != "ok.mp4" {
		t.Fatalf("expected only ok.mp4, got %+v", m.entries)
	}
}
