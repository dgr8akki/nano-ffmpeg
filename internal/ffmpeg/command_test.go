package ffmpeg

import (
	"strings"
	"testing"
)

func TestNewCommand(t *testing.T) {
	cmd := NewCommand("/usr/bin/ffmpeg", "input.mp4", "output.mkv")
	if cmd.FFmpegPath != "/usr/bin/ffmpeg" {
		t.Errorf("unexpected path: %s", cmd.FFmpegPath)
	}
	if cmd.Input != "input.mp4" {
		t.Errorf("unexpected input: %s", cmd.Input)
	}
	if cmd.Output != "output.mkv" {
		t.Errorf("unexpected output: %s", cmd.Output)
	}
}

func TestBuildBasic(t *testing.T) {
	cmd := NewCommand("/usr/bin/ffmpeg", "in.mp4", "out.mp4")
	args := cmd.Build()

	expected := []string{"-y", "-i", "in.mp4", "out.mp4"}
	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(args), args)
	}
	for i, a := range args {
		if a != expected[i] {
			t.Errorf("arg[%d]: expected %q, got %q", i, expected[i], a)
		}
	}
}

func TestBuildConvert(t *testing.T) {
	cmd := NewCommand("/usr/bin/ffmpeg", "in.mkv", "out.mp4")
	cmd.SetVideoCodec("libx264")
	cmd.SetCRF(23)
	cmd.SetPreset("medium")
	cmd.SetAudioCodec("aac")

	args := cmd.Build()
	str := strings.Join(args, " ")

	if !strings.Contains(str, "-c:v libx264") {
		t.Errorf("missing video codec: %s", str)
	}
	if !strings.Contains(str, "-crf 23") {
		t.Errorf("missing crf: %s", str)
	}
	if !strings.Contains(str, "-preset medium") {
		t.Errorf("missing preset: %s", str)
	}
	if !strings.Contains(str, "-c:a aac") {
		t.Errorf("missing audio codec: %s", str)
	}
}

func TestBuildTrim(t *testing.T) {
	cmd := NewCommand("/usr/bin/ffmpeg", "in.mp4", "out.mp4")
	cmd.SetStartTime("00:01:00")
	cmd.SetEndTime("00:02:00")
	cmd.StreamCopy()

	args := cmd.Build()
	str := strings.Join(args, " ")

	if !strings.Contains(str, "-ss 00:01:00") {
		t.Errorf("missing start time: %s", str)
	}
	if !strings.Contains(str, "-to 00:02:00") {
		t.Errorf("missing end time: %s", str)
	}
	if !strings.Contains(str, "-c copy") {
		t.Errorf("missing stream copy: %s", str)
	}
}

func TestBuildExtractAudio(t *testing.T) {
	cmd := NewCommand("/usr/bin/ffmpeg", "in.mp4", "out.mp3")
	cmd.NoVideo()
	cmd.SetAudioCodec("libmp3lame")
	cmd.SetAudioBitrate("320k")

	args := cmd.Build()
	str := strings.Join(args, " ")

	if !strings.Contains(str, "-vn") {
		t.Errorf("missing no-video flag: %s", str)
	}
	if !strings.Contains(str, "-c:a libmp3lame") {
		t.Errorf("missing audio codec: %s", str)
	}
	if !strings.Contains(str, "-b:a 320k") {
		t.Errorf("missing audio bitrate: %s", str)
	}
}

func TestBuildResize(t *testing.T) {
	cmd := NewCommand("/usr/bin/ffmpeg", "in.mp4", "out.mp4")
	cmd.SetScaleHeight(720)
	cmd.SetVideoCodec("libx264")

	args := cmd.Build()
	str := strings.Join(args, " ")

	if !strings.Contains(str, "-vf scale=-2:720") {
		t.Errorf("missing scale filter: %s", str)
	}
}

func TestCommandString(t *testing.T) {
	cmd := NewCommand("/usr/bin/ffmpeg", "my file.mp4", "output.mp4")
	cmd.SetVideoCodec("libx264")

	s := cmd.String()
	if !strings.Contains(s, `"my file.mp4"`) {
		t.Errorf("path with spaces should be quoted: %s", s)
	}
}

func TestSetPresetForCodec(t *testing.T) {
	tests := []struct {
		name     string
		codec    string
		preset   string
		expected string
	}{
		{"libsvtav1 maps slow to 4", "libsvtav1", "slow", "-preset 4"},
		{"libsvtav1 maps medium to 6", "libsvtav1", "medium", "-preset 6"},
		{"libsvtav1 maps fast to 9", "libsvtav1", "fast", "-preset 9"},
		{"libsvtav1 maps ultrafast to 12", "libsvtav1", "ultrafast", "-preset 12"},
		{"libsvtav1 is case insensitive", "libsvtav1", "Slow", "-preset 4"},
		{"libsvtav1 passes through numeric preset", "libsvtav1", "8", "-preset 8"},
		{"libsvtav1 falls back to medium for unknown", "libsvtav1", "bogus", "-preset 6"},
		{"libx264 keeps string preset", "libx264", "slow", "-preset slow"},
		{"libx265 keeps string preset", "libx265", "medium", "-preset medium"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewCommand("/usr/bin/ffmpeg", "in.mp4", "out.mkv")
			cmd.SetVideoCodec(tc.codec)
			cmd.SetPresetForCodec(tc.codec, tc.preset)

			str := strings.Join(cmd.Build(), " ")
			if !strings.Contains(str, tc.expected) {
				t.Errorf("expected %q in args, got: %s", tc.expected, str)
			}
		})
	}
}

func TestNoOverwrite(t *testing.T) {
	cmd := NewCommand("/usr/bin/ffmpeg", "in.mp4", "out.mp4")
	cmd.Overwrite = false
	args := cmd.Build()
	for _, a := range args {
		if a == "-y" {
			t.Error("-y flag should not be present when Overwrite is false")
		}
	}
}
