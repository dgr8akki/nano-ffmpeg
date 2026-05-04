package ffmpeg

import (
	"strings"
	"testing"
)

func TestTwoPassSupported(t *testing.T) {
	for codec, want := range map[string]bool{
		"libx264":    true,
		"libx265":    true,
		"libvpx-vp9": true,
		"libsvtav1":  false,
		"":           false,
	} {
		if got := TwoPassSupported(codec); got != want {
			t.Errorf("TwoPassSupported(%q) = %v, want %v", codec, got, want)
		}
	}
}

func TestBuildTwoPassCommands(t *testing.T) {
	cmds := BuildTwoPassCommands("ffmpeg", "in.mp4", "out.mp4", "libx264", "medium", "2500k", "/tmp/stats")
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}
	a1 := strings.Join(cmds[0].Build(), " ")
	a2 := strings.Join(cmds[1].Build(), " ")
	if !strings.Contains(a1, "-pass 1") || !strings.Contains(a1, "-passlogfile /tmp/stats") {
		t.Errorf("pass 1 missing flags: %s", a1)
	}
	if !strings.Contains(a1, "-an") || !strings.Contains(a1, "-f null") {
		t.Errorf("pass 1 should be silent/null: %s", a1)
	}
	if !strings.Contains(a2, "-pass 2") || !strings.Contains(a2, "out.mp4") {
		t.Errorf("pass 2 missing flags: %s", a2)
	}
	if len(cmds[1].Cleanup) == 0 {
		t.Error("expected cleanup paths on pass 2")
	}
}
