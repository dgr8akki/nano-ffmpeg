//go:build e2e

// End-to-end check: builds every operation's command through the real
// settings model, runs it against generated sample media with the local
// ffmpeg, and verifies the output with ffprobe.
//
//	go test -tags e2e ./internal/screens/settings/ -run E2E -v
package settings

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dgr8akki/nano-ffmpeg/internal/ffmpeg"
	"github.com/dgr8akki/nano-ffmpeg/internal/screens/operations"
)

var ops = map[operations.OperationID]string{}

func init() {
	for _, op := range operations.AllOperations {
		ops[op.ID] = op.Name
	}
}

type fixtures struct {
	dir    string
	ffmpeg string
	probe  string
}

func gen(t *testing.T, ff, out string, args ...string) {
	t.Helper()
	full := append([]string{"-hide_banner", "-loglevel", "error", "-y"}, args...)
	full = append(full, out)
	if b, err := exec.Command(ff, full...).CombinedOutput(); err != nil {
		t.Fatalf("fixture %s: %v\n%s", out, err, b)
	}
}

func makeFixtures(t *testing.T) fixtures {
	info, err := ffmpeg.Detect()
	if err != nil {
		t.Skip("ffmpeg not installed")
	}
	dir := t.TempDir()
	ff := info.FFmpegPath
	video := func(name, size, dur string, audio bool, extra ...string) {
		args := []string{"-f", "lavfi", "-i", "testsrc2=s=" + size + ":r=30:d=" + dur}
		if audio {
			args = append(args, "-f", "lavfi", "-i", "sine=f=440:d="+dur)
		}
		args = append(args, extra...)
		gen(t, ff, filepath.Join(dir, name), args...)
	}
	video("normal.mp4", "1280x720", "10", true, "-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p", "-c:a", "aac")
	video("noaudio.mp4", "1280x720", "6", false, "-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p")
	video("short.mp4", "640x360", "2", true, "-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p", "-c:a", "aac")
	video("odd.mkv", "641x361", "4", true, "-c:v", "ffv1", "-c:a", "flac")
	video("portrait.mov", "720x1280", "4", true, "-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p", "-c:a", "aac")
	gen(t, ff, filepath.Join(dir, "audio.mp3"), "-f", "lavfi", "-i", "sine=f=440:d=5", "-c:a", "libmp3lame")

	srt := filepath.Join(dir, "subs.srt")
	os.WriteFile(srt, []byte("1\n00:00:00,500 --> 00:00:03,000\nHello world\n\n2\n00:00:04,000 --> 00:00:08,000\nSecond line\n"), 0644)
	gen(t, ff, filepath.Join(dir, "subs.mkv"), "-i", filepath.Join(dir, "normal.mp4"), "-i", srt, "-map", "0", "-map", "1", "-c", "copy", "-c:s", "srt", "-metadata:s:s:0", "language=eng")
	gen(t, ff, filepath.Join(dir, "subs.mp4"), "-i", filepath.Join(dir, "normal.mp4"), "-i", srt, "-map", "0", "-map", "1", "-c", "copy", "-c:s", "mov_text")

	normal, _ := os.ReadFile(filepath.Join(dir, "normal.mp4"))
	os.WriteFile(filepath.Join(dir, "my clip's [final] 100%.mp4"), normal, 0644)
	os.WriteFile(filepath.Join(dir, "vidéo ✓.mp4"), normal, 0644)

	for _, sub := range []string{"merge", "merge_spaces", "merge_mismatch", "merge_quotes"} {
		os.Mkdir(filepath.Join(dir, sub), 0755)
	}
	for _, n := range []string{"a.mp4", "b.mp4", "c.mp4"} {
		os.WriteFile(filepath.Join(dir, "merge", n), normal, 0644)
	}
	os.WriteFile(filepath.Join(dir, "merge_spaces", "part one.mp4"), normal, 0644)
	os.WriteFile(filepath.Join(dir, "merge_spaces", "part two.mp4"), normal, 0644)
	os.WriteFile(filepath.Join(dir, "merge_quotes", "it's one.mp4"), normal, 0644)
	os.WriteFile(filepath.Join(dir, "merge_quotes", "vidéo ✓ 100%.mp4"), normal, 0644)
	os.WriteFile(filepath.Join(dir, "merge_mismatch", "a.mp4"), normal, 0644)
	short, _ := os.ReadFile(filepath.Join(dir, "short.mp4"))
	os.WriteFile(filepath.Join(dir, "merge_mismatch", "b.mp4"), short, 0644)

	return fixtures{dir: dir, ffmpeg: ff, probe: info.FFprobePath}
}

func (m *Model) set(label, value string) {
	for i := range m.fields {
		f := &m.fields[i]
		if f.Label != label {
			continue
		}
		switch f.Type {
		case FieldSelect:
			for j, o := range f.Options {
				if o.Value == value {
					f.Selected, f.Value = j, value
					return
				}
			}
			panic("no option " + value + " for " + label)
		case FieldToggle:
			f.Enabled = value == "true"
			f.Value = value
		default:
			f.Value = value
		}
		return
	}
	panic("no field " + label)
}

type tc struct {
	name   string
	op     operations.OperationID
	input  string
	fields [][2]string
	// expectations; empty means "don't care"
	wantVideo, wantAudio, wantSubs *bool
	wantDur                        float64 // approx seconds, 0 = skip
	blocked                        bool    // settings screen must refuse to run
}

func b(v bool) *bool { return &v }

func runCase(t *testing.T, fx fixtures, c tc) {
	in := filepath.Join(fx.dir, c.input)
	probe, err := ffmpeg.Probe(fx.probe, in)
	if err != nil {
		t.Fatalf("probe input: %v", err)
	}
	m := New(c.op, ops[c.op], in, probe, fx.ffmpeg)
	for _, kv := range c.fields {
		m.set(kv[0], kv[1])
	}
	if verr := m.validationError(); verr != "" || c.blocked {
		if !c.blocked {
			t.Fatalf("unexpectedly blocked: %s", verr)
		}
		if verr == "" {
			t.Fatalf("should have been blocked by validation")
		}
		t.Logf("blocked as expected: %s", verr)
		return
	}
	cmds := m.buildCommands()
	out := cmds[len(cmds)-1].Output
	os.Remove(out)

	total := probe.Format.Duration
	if d := m.expectedDuration(); d > 0 {
		total = d
	}
	parser := ffmpeg.NewProgressParser(total)
	var last *ffmpeg.Progress
	var tail []string
	for _, cmd := range cmds {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		ex := exec.CommandContext(ctx, cmd.FFmpegPath, cmd.Build()...)
		stderr, _ := ex.StderrPipe()
		ex.Start()
		sc := bufio.NewScanner(stderr)
		sc.Split(scanCR)
		for sc.Scan() {
			line := sc.Text()
			if p := parser.Parse(line); p != nil {
				last = p
			}
			tail = append(tail, line)
			if len(tail) > 8 {
				tail = tail[1:]
			}
		}
		err := ex.Wait()
		timedOut := ctx.Err() != nil
		cancel()
		if timedOut {
			t.Fatalf("TIMEOUT (ran >90s)\n$ %s", cmd.String())
		}
		if err != nil {
			t.Fatalf("ffmpeg failed: %v\n$ %s\n%s", err, cmd.String(), strings.Join(tail, "\n"))
		}
	}
	check1 := out
	if strings.Contains(out, "%03d") {
		check1 = fmt.Sprintf(out, 1)
		if _, err := os.Stat(fmt.Sprintf(out, 2)); err != nil {
			t.Errorf("image sequence wrote only one frame")
		}
	}
	st, err := os.Stat(check1)
	if err != nil || st.Size() == 0 {
		t.Fatalf("no output at %s", check1)
	}
	if strings.HasSuffix(out, ".png") {
		// image muxer reports size=N/A
	} else if last == nil || last.Size == 0 {
		t.Errorf("progress parser saw no output size (last=%+v)", last)
	} else if c.wantDur > 0 && last.Percent < 95 {
		t.Errorf("progress bar ended at %.0f%%", last.Percent)
	}
	if strings.HasSuffix(out, ".png") {
		return
	}
	res, err := ffmpeg.Probe(fx.probe, out)
	if err != nil {
		t.Fatalf("probe output: %v", err)
	}
	check := func(what string, want *bool, got bool) {
		if want != nil && *want != got {
			t.Errorf("%s stream: want %v got %v", what, *want, got)
		}
	}
	check("video", c.wantVideo, res.VideoStream() != nil)
	check("audio", c.wantAudio, res.AudioStream() != nil)
	check("subtitle", c.wantSubs, len(res.SubtitleStreams()) > 0)
	if c.wantDur > 0 {
		d := res.Format.Duration
		if d < c.wantDur*0.85 || d > c.wantDur*1.15+0.3 {
			t.Errorf("duration: want ~%.1fs got %.2fs", c.wantDur, d)
		}
	}
	// sidecar files left next to the input?
	ents, _ := os.ReadDir(filepath.Dir(in))
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), ".nano-ffmpeg") {
			t.Errorf("leftover temp file %s next to the input", e.Name())
		}
	}
}

func scanCR(data []byte, atEOF bool) (int, []byte, error) {
	for i, c := range data {
		if c == '\n' || c == '\r' {
			return i + 1, data[:i], nil
		}
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func TestE2E(t *testing.T) {
	fx := makeFixtures(t)
	T, F := b(true), b(false)
	var cases []tc

	// Convert: every format x codec, ultrafast to keep it quick.
	for _, format := range []string{"mp4", "mkv", "webm", "avi", "mov"} {
		for _, codec := range []string{"libx264", "libx265", "libsvtav1", "libvpx-vp9"} {
			cases = append(cases, tc{name: "convert/" + format + "/" + codec, op: operations.OpConvert, input: "short.mp4",
				fields: [][2]string{{"Format", format}, {"Codec", codec}, {"Preset", "ultrafast"}}, wantVideo: T, wantAudio: T, wantDur: 2})
		}
	}
	for _, a := range []string{"aac", "libmp3lame", "libopus"} {
		cases = append(cases, tc{name: "convert/mp4/audio-" + a, op: operations.OpConvert, input: "short.mp4",
			fields: [][2]string{{"Audio", a}, {"Preset", "ultrafast"}}, wantVideo: T, wantAudio: T, wantDur: 2})
	}
	cases = append(cases,
		tc{name: "convert/defaults", op: operations.OpConvert, input: "normal.mp4", wantVideo: T, wantAudio: T, wantDur: 10},
		tc{name: "convert/av1-slow", op: operations.OpConvert, input: "short.mp4", fields: [][2]string{{"Codec", "libsvtav1"}, {"Preset", "slow"}}, wantVideo: T, wantDur: 2},
		tc{name: "convert/odd-dims-h264", op: operations.OpConvert, input: "odd.mkv", fields: [][2]string{{"Preset", "ultrafast"}}, wantVideo: T, wantDur: 4},
		tc{name: "convert/odd-dims-x265", op: operations.OpConvert, input: "odd.mkv", fields: [][2]string{{"Codec", "libx265"}, {"Preset", "ultrafast"}}, wantVideo: T, wantDur: 4},
		tc{name: "convert/audio-only-input", op: operations.OpConvert, input: "audio.mp3", wantAudio: T, wantDur: 5},
		tc{name: "convert/special-chars-name", op: operations.OpConvert, input: "my clip's [final] 100%.mp4", fields: [][2]string{{"Preset", "ultrafast"}}, wantVideo: T, wantDur: 10},
		tc{name: "convert/unicode-name", op: operations.OpConvert, input: "vidéo ✓.mp4", fields: [][2]string{{"Preset", "ultrafast"}}, wantVideo: T, wantDur: 10},
	)

	for _, f := range []string{"mp3", "m4a", "flac", "wav", "ogg", "opus"} {
		cases = append(cases, tc{name: "extract/" + f, op: operations.OpExtractAudio, input: "normal.mp4", fields: [][2]string{{"Format", f}}, wantVideo: F, wantAudio: T, wantDur: 10})
	}
	cases = append(cases, tc{name: "extract/no-audio-input", op: operations.OpExtractAudio, input: "noaudio.mp4", blocked: true})

	for _, r := range []string{"2160", "1080", "720", "480", "360"} {
		cases = append(cases, tc{name: "resize/" + r, op: operations.OpResize, input: "short.mp4", fields: [][2]string{{"Resolution", r}}, wantVideo: T, wantAudio: T, wantDur: 2})
	}
	cases = append(cases,
		tc{name: "resize/odd-input", op: operations.OpResize, input: "odd.mkv", fields: [][2]string{{"Resolution", "360"}}, wantVideo: T, wantDur: 4},
		tc{name: "resize/portrait", op: operations.OpResize, input: "portrait.mov", fields: [][2]string{{"Resolution", "480"}}, wantVideo: T, wantDur: 4},
		tc{name: "resize/x265", op: operations.OpResize, input: "short.mp4", fields: [][2]string{{"Codec", "libx265"}}, wantVideo: T, wantDur: 2},
	)

	cases = append(cases,
		tc{name: "trim/defaults", op: operations.OpTrim, input: "normal.mp4", wantVideo: T, wantAudio: T, wantDur: 10},
		tc{name: "trim/2-6-lossless", op: operations.OpTrim, input: "normal.mp4", fields: [][2]string{{"Start Time", "00:00:02"}, {"End Time", "00:00:06"}}, wantVideo: T, wantDur: 4},
		tc{name: "trim/2-6-reencode", op: operations.OpTrim, input: "normal.mp4", fields: [][2]string{{"Start Time", "00:00:02"}, {"End Time", "00:00:06"}, {"Lossless Cut", "false"}}, wantVideo: T, wantDur: 4},
		tc{name: "trim/seconds-format", op: operations.OpTrim, input: "normal.mp4", fields: [][2]string{{"Start Time", "1.5"}, {"End Time", "3"}, {"Lossless Cut", "false"}}, wantVideo: T, wantDur: 1.5},
		tc{name: "trim/empty-end", op: operations.OpTrim, input: "normal.mp4", fields: [][2]string{{"Start Time", "00:00:08"}, {"End Time", ""}, {"Lossless Cut", "false"}}, wantVideo: T, wantDur: 2},
		tc{name: "trim/end-before-start", op: operations.OpTrim, input: "normal.mp4", fields: [][2]string{{"Start Time", "00:00:06"}, {"End Time", "00:00:02"}}, blocked: true},
		tc{name: "trim/garbage-time", op: operations.OpTrim, input: "normal.mp4", fields: [][2]string{{"Start Time", "abc"}}, blocked: true},
		tc{name: "trim/start-past-end", op: operations.OpTrim, input: "short.mp4", fields: [][2]string{{"Start Time", "00:00:30"}, {"End Time", "00:00:40"}}, blocked: true},
		tc{name: "trim/mm:ss", op: operations.OpTrim, input: "normal.mp4", fields: [][2]string{{"Start Time", "0:03"}, {"End Time", "0:07.5"}, {"Lossless Cut", "false"}}, wantVideo: T, wantDur: 4.5},
	)

	for _, codec := range []string{"libx264", "libx265", "libsvtav1"} {
		for _, p := range []string{"slow", "medium", "fast"} {
			cases = append(cases, tc{name: "compress/" + codec + "/" + p, op: operations.OpCompress, input: "short.mp4", fields: [][2]string{{"Codec", codec}, {"Preset", p}}, wantVideo: T, wantAudio: T, wantDur: 2})
		}
	}

	cases = append(cases,
		tc{name: "merge/copy", op: operations.OpMerge, input: "merge/a.mp4", wantVideo: T, wantAudio: T, wantDur: 30},
		tc{name: "merge/quote-and-accent-names", op: operations.OpMerge, input: "merge_quotes/it's one.mp4", wantVideo: T, wantAudio: T, wantDur: 20},
		tc{name: "merge/copy-rerun", op: operations.OpMerge, input: "merge/a.mp4", wantVideo: T, wantAudio: T, wantDur: 30},
		tc{name: "merge/reencode", op: operations.OpMerge, input: "merge/a.mp4", fields: [][2]string{{"Merge Mode", "reencode"}}, wantVideo: T, wantAudio: T, wantDur: 30},
		tc{name: "merge/spaces-in-names", op: operations.OpMerge, input: "merge_spaces/part one.mp4", wantVideo: T, wantDur: 20},
		tc{name: "merge/mismatched-reencode", op: operations.OpMerge, input: "merge_mismatch/a.mp4", fields: [][2]string{{"Merge Mode", "reencode"}}, wantVideo: T, wantDur: 12},
	)

	cases = append(cases,
		tc{name: "subs/burn-mkv", op: operations.OpSubtitles, input: "subs.mkv", wantVideo: T, wantDur: 10, blocked: !ffmpegHas(fx.ffmpeg, "-filters", "subtitles")},
		tc{name: "subs/embed-mkv", op: operations.OpSubtitles, input: "subs.mkv", fields: [][2]string{{"Subtitle Mode", "embed"}}, wantVideo: T, wantSubs: T, wantDur: 10},
		tc{name: "subs/embed-mp4", op: operations.OpSubtitles, input: "subs.mp4", fields: [][2]string{{"Subtitle Mode", "embed"}}, wantVideo: T, wantSubs: T, wantDur: 10},
		tc{name: "subs/burn-no-subs", op: operations.OpSubtitles, input: "short.mp4", blocked: true},
	)

	for _, pos := range []string{"top-left", "top-right", "bottom-left", "bottom-right", "center"} {
		cases = append(cases, tc{name: "watermark/" + pos, op: operations.OpWatermark, input: "short.mp4", fields: [][2]string{{"Position", pos}}, wantVideo: T, wantAudio: T, wantDur: 2})
	}
	cases = append(cases,
		tc{name: "watermark/no-audio", op: operations.OpWatermark, input: "noaudio.mp4", wantVideo: T, wantDur: 6},
		tc{name: "watermark/small-video-large-mark", op: operations.OpWatermark, input: "odd.mkv", fields: [][2]string{{"Size", "320x120"}}, wantVideo: T, wantDur: 4},
	)

	cases = append(cases,
		tc{name: "gif/defaults", op: operations.OpGIF, input: "normal.mp4", wantVideo: T, wantDur: 5},
		tc{name: "gif/start-3", op: operations.OpGIF, input: "normal.mp4", fields: [][2]string{{"Start Time", "00:00:03"}, {"FPS", "10"}, {"Width", "320"}}, wantVideo: T, wantDur: 5},
		tc{name: "gif/short-input", op: operations.OpGIF, input: "short.mp4", wantVideo: T, wantDur: 2},
		tc{name: "gif/empty-duration", op: operations.OpGIF, input: "short.mp4", fields: [][2]string{{"Duration", ""}}, wantVideo: T, wantDur: 2},
	)

	cases = append(cases,
		tc{name: "thumbs/single", op: operations.OpThumbnails, input: "normal.mp4"},
		tc{name: "thumbs/single-short-clip-default", op: operations.OpThumbnails, input: "short.mp4"},
		tc{name: "thumbs/single-past-end", op: operations.OpThumbnails, input: "short.mp4", fields: [][2]string{{"Timestamp", "00:00:05"}}, blocked: true},
		tc{name: "thumbs/grid", op: operations.OpThumbnails, input: "normal.mp4", fields: [][2]string{{"Mode", "grid"}}},
		tc{name: "thumbs/grid-short", op: operations.OpThumbnails, input: "short.mp4", fields: [][2]string{{"Mode", "grid"}}},
		tc{name: "thumbs/interval", op: operations.OpThumbnails, input: "normal.mp4", fields: [][2]string{{"Mode", "interval"}}},
		tc{name: "thumbs/interval-percent-name", op: operations.OpThumbnails, input: "my clip's [final] 100%.mp4", fields: [][2]string{{"Mode", "interval"}}},
		tc{name: "thumbs/single-percent-name", op: operations.OpThumbnails, input: "my clip's [final] 100%.mp4"},
		tc{name: "gif/percent-name", op: operations.OpGIF, input: "my clip's [final] 100%.mp4", wantVideo: T, wantDur: 5},
	)

	for _, op := range []string{"normalize", "up", "down", "fade", "remove"} {
		cases = append(cases, tc{name: "audio/" + op, op: operations.OpAudio, input: "normal.mp4", fields: [][2]string{{"Operation", op}}, wantVideo: T, wantAudio: b(op != "remove"), wantDur: 10})
	}
	cases = append(cases,
		tc{name: "audio/fade-short", op: operations.OpAudio, input: "short.mp4", fields: [][2]string{{"Operation", "fade"}}, wantVideo: T, wantDur: 2},
		tc{name: "audio/normalize-no-audio", op: operations.OpAudio, input: "noaudio.mp4", blocked: true},
		tc{name: "audio/normalize-audio-only", op: operations.OpAudio, input: "audio.mp3", wantAudio: T, wantDur: 5},
	)

	for _, f := range []string{"vidstab", "yadif", "speed2", "speed05", "rotate90", "hflip", "vflip"} {
		dur := 2.0
		switch f {
		case "speed2":
			dur = 1
		case "speed05":
			dur = 4
		}
		cases = append(cases, tc{name: "filters/" + f, op: operations.OpFilters, input: "short.mp4", fields: [][2]string{{"Filter", f}}, wantVideo: T, wantAudio: T, wantDur: dur})
	}
	cases = append(cases,
		tc{name: "filters/speed2-no-audio", op: operations.OpFilters, input: "noaudio.mp4", fields: [][2]string{{"Filter", "speed2"}}, wantVideo: T, wantDur: 3},
		tc{name: "filters/rotate-audio-only", op: operations.OpFilters, input: "audio.mp3", fields: [][2]string{{"Filter", "rotate90"}}, blocked: true},
	)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { runCase(t, fx, c) })
	}
}
