package settings

import (
	"strings"
	"testing"

	"github.com/dgr8akki/nano-ffmpeg/internal/screens/operations"
)

func newCompressModel() *Model {
	return New(operations.OpCompress, "Compress", "/tmp/in.mp4", nil, "ffmpeg")
}

func TestCompress_DefaultIsCRFOnePass(t *testing.T) {
	m := newCompressModel()
	cmds := m.buildCommands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 cmd in CRF mode, got %d", len(cmds))
	}
	args := strings.Join(cmds[0].Build(), " ")
	if !strings.Contains(args, "-crf 23") {
		t.Errorf("expected -crf 23, got: %s", args)
	}
}

func TestCompress_ToggleSwapsQualityForBitrate(t *testing.T) {
	m := newCompressModel()
	for i, f := range m.fields {
		if f.Label == "Two-Pass" {
			m.cursor = i
			m.adjustField(0)
			break
		}
	}
	hasBitrate := false
	hasQuality := false
	for _, f := range m.fields {
		if f.Label == "Target Bitrate" {
			hasBitrate = true
		}
		if f.Label == "Quality" {
			hasQuality = true
		}
	}
	if !hasBitrate || hasQuality {
		t.Fatalf("after enabling two-pass: hasBitrate=%v hasQuality=%v", hasBitrate, hasQuality)
	}

	cmds := m.buildCommands()
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands for two-pass, got %d", len(cmds))
	}
	a1 := strings.Join(cmds[0].Build(), " ")
	a2 := strings.Join(cmds[1].Build(), " ")
	if !strings.Contains(a1, "-pass 1") || !strings.Contains(a2, "-pass 2") {
		t.Errorf("pass flags missing:\n1: %s\n2: %s", a1, a2)
	}
	if !strings.Contains(a2, "-b:v 2500k") {
		t.Errorf("expected default 2500k bitrate: %s", a2)
	}
}

func TestCompress_UnsupportedCodecForcesToggleOff(t *testing.T) {
	m := newCompressModel()
	for i, f := range m.fields {
		if f.Label == "Two-Pass" {
			m.cursor = i
			m.adjustField(0)
			break
		}
	}
	for i, f := range m.fields {
		if f.Label == "Codec" {
			m.cursor = i
			for f.Options[m.fields[i].Selected].Value != "libsvtav1" {
				m.adjustField(1)
				f = m.fields[i]
			}
			break
		}
	}
	if m.fieldEnabled("Two-Pass") {
		t.Fatal("two-pass should auto-disable for libsvtav1")
	}
	if notice := m.fallbackNotice(); !strings.Contains(notice, "Two-pass") {
		t.Errorf("expected unsupported-codec notice, got %q", notice)
	}
}
