package ffmpeg

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeFakeBinary writes a shell script that prints the given stdout content.
func writeFakeBinary(t *testing.T, stdout string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fake")
	// Use a heredoc marker that won't appear in the payload.
	script := "#!/bin/sh\ncat <<'__END__'\n" + stdout + "\n__END__\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return path
}

func TestParseCodecs_FromFakeFFmpeg(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX shell")
	}

	// Representative ffmpeg -codecs output (column meanings from ffmpeg docs):
	//  D....... = Decoding supported
	//  .E...... = Encoding supported
	//  ..V..... = Video codec
	//  ..A..... = Audio codec
	//  ..S..... = Subtitle codec
	//  ...I.... = Intra frame-only codec
	//  ....L... = Lossy compression
	//  .....S.. = Lossless compression
	output := ` D.V.L. h264                 H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10
 DEV.L. libx264              libx264 H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10 (encoders: libx264 libx264rgb )
 DEA.L. aac                  AAC (Advanced Audio Coding)
 DES..S mov_text             MOV text`
	bin := writeFakeBinary(t, output)

	codecs, err := parseCodecs(bin)
	if err != nil {
		t.Fatalf("parseCodecs: %v", err)
	}

	byName := map[string]Codec{}
	for _, c := range codecs {
		byName[c.Name] = c
	}

	h264, ok := byName["h264"]
	if !ok {
		t.Fatal("expected h264 codec")
	}
	if !h264.Decoding || h264.Encoding {
		t.Fatalf("h264: expected decode-only, got %+v", h264)
	}
	if h264.Type != "video" {
		t.Fatalf("h264 type: got %q, want video", h264.Type)
	}

	libx264, ok := byName["libx264"]
	if !ok {
		t.Fatal("expected libx264")
	}
	if !libx264.Encoding || !libx264.Decoding {
		t.Fatalf("libx264 should support decode+encode, got %+v", libx264)
	}

	aac, ok := byName["aac"]
	if !ok {
		t.Fatal("expected aac")
	}
	if aac.Type != "audio" {
		t.Fatalf("aac type: got %q, want audio", aac.Type)
	}

	if sub, ok := byName["mov_text"]; !ok {
		t.Fatal("expected mov_text")
	} else if sub.Type != "subtitle" {
		t.Fatalf("mov_text type: got %q, want subtitle", sub.Type)
	}
}

func TestParseFormats_SkipsHeader(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX shell")
	}

	output := `File formats:
 D. = Demuxing supported
 .E = Muxing supported
 ---
 DE mov              QuickTime / MOV
 D  mpeg             MPEG-1 Systems
  E md5              MD5 testing`
	bin := writeFakeBinary(t, output)

	formats, err := parseFormats(bin)
	if err != nil {
		t.Fatalf("parseFormats: %v", err)
	}
	if len(formats) != 3 {
		t.Fatalf("expected 3 formats, got %d: %+v", len(formats), formats)
	}

	byName := map[string]Format{}
	for _, f := range formats {
		byName[f.Name] = f
	}
	if f := byName["mov"]; !f.Demux || !f.Mux {
		t.Fatalf("mov should support both demux+mux, got %+v", f)
	}
	if f := byName["mpeg"]; !f.Demux || f.Mux {
		t.Fatalf("mpeg should be demux-only, got %+v", f)
	}
	if f := byName["md5"]; f.Demux || !f.Mux {
		t.Fatalf("md5 should be mux-only, got %+v", f)
	}
}

func TestParseFilters_StripsHeader(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX shell")
	}

	output := `Filters:
 T.. = Timeline support
 .S. = Slice threading
 ..C = Command support
 A = Audio input/output
 V = Video input/output
 N = Dynamic number and/or type of input/output
 | = Source or sink filter
------
 T.. scale            V->V       Scale the input video size.
 ... vidstabdetect    V->V       Analyze for stabilization.
 ... loudnorm         A->A       EBU R128 loudness normalization.`
	bin := writeFakeBinary(t, output)

	filters, err := parseFilters(bin)
	if err != nil {
		t.Fatalf("parseFilters: %v", err)
	}

	want := []string{"scale", "vidstabdetect", "loudnorm"}
	for _, name := range want {
		found := false
		for _, f := range filters {
			if f == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected filter %q in result: %v", name, filters)
		}
	}
}

func TestParseHWAccels_IgnoresBlankLines(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX shell")
	}

	output := `Hardware acceleration methods:
videotoolbox
cuda

`
	bin := writeFakeBinary(t, output)

	accels, err := parseHWAccels(bin)
	if err != nil {
		t.Fatalf("parseHWAccels: %v", err)
	}
	if len(accels) != 2 {
		t.Fatalf("expected 2 accels, got %d: %v", len(accels), accels)
	}
	if accels[0] != "videotoolbox" || accels[1] != "cuda" {
		t.Fatalf("unexpected accels: %v", accels)
	}
}

func TestCacheCapabilities_RoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	original := &Capabilities{
		Version: "6.1",
		Codecs: []Codec{
			{Name: "libx264", Encoding: true, Type: "video"},
		},
		Formats: []Format{
			{Name: "mp4", Mux: true, Demux: true},
		},
		Filters:  []string{"scale"},
		HWAccels: []string{"videotoolbox"},
	}

	if err := cacheCapabilities(original); err != nil {
		t.Fatalf("cacheCapabilities: %v", err)
	}

	loaded, err := loadCachedCapabilities("6.1")
	if err != nil {
		t.Fatalf("loadCachedCapabilities: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected cached capabilities")
	}
	if loaded.Version != original.Version {
		t.Fatalf("version: got %q, want %q", loaded.Version, original.Version)
	}
	if len(loaded.Codecs) != 1 || loaded.Codecs[0].Name != "libx264" {
		t.Fatalf("codecs not restored: %+v", loaded.Codecs)
	}
}

func TestLoadCachedCapabilities_StaleVersion(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := cacheCapabilities(&Capabilities{Version: "6.1"}); err != nil {
		t.Fatalf("cacheCapabilities: %v", err)
	}

	loaded, err := loadCachedCapabilities("6.2")
	if err != nil {
		t.Fatalf("loadCachedCapabilities: %v", err)
	}
	if loaded != nil {
		t.Fatalf("expected stale cache to return nil, got %+v", loaded)
	}
}

func TestLoadCachedCapabilities_MissingFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	loaded, err := loadCachedCapabilities("6.1")
	if err == nil {
		t.Fatal("expected error for missing cache file")
	}
	if loaded != nil {
		t.Fatalf("expected nil result, got %+v", loaded)
	}
}

func TestConfigAndCachePath_UsesHomeDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if got := configDir(); !strings.HasPrefix(got, home) {
		t.Fatalf("expected configDir to live under %q, got %q", home, got)
	}
	if got := cachePath(); filepath.Base(got) != "capabilities.json" {
		t.Fatalf("unexpected cache filename: %q", got)
	}
}

// ffmpeg 8+ dropped the command-support column: flags are two chars wide.
func TestParseFilters_FFmpeg8TwoColumnFlags(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX shell")
	}
	bin := writeFakeBinary(t, `Filters:
  T.. = Timeline support
  ------
 TS aap               AA->A      Apply Affine Projection algorithm.
 .. deshake           V->V       Stabilize shaky video.
 .. movie             |->N       Read from a movie source.`)
	filters, err := parseFilters(bin)
	if err != nil {
		t.Fatalf("parseFilters: %v", err)
	}
	if strings.Join(filters, ",") != "aap,deshake,movie" {
		t.Fatalf("got %v", filters)
	}
}
