package ffmpeg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// Info holds detected ffmpeg installation details.
type Info struct {
	FFmpegPath  string
	FFprobePath string
	Version     string
	BuildConfig string
}

// Detect finds ffmpeg and ffprobe on the system and returns installation info.
func Detect() (*Info, error) {
	ffmpegPath, err := findBinary("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg binary not found: %w", err)
	}

	ffprobePath, err := findBinary("ffprobe")
	if err != nil {
		return nil, fmt.Errorf("ffprobe binary not found: %w", err)
	}

	version, buildConfig, err := parseVersion(ffmpegPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ffmpeg version: %w", err)
	}

	return &Info{
		FFmpegPath:  ffmpegPath,
		FFprobePath: ffprobePath,
		Version:     version,
		BuildConfig: buildConfig,
	}, nil
}

func findBinary(name string) (string, error) {
	// Try PATH first
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}

	if p, err := findNextToExecutable(name); err == nil {
		return p, nil
	}

	// Fallback to common and keg-only Homebrew locations.
	for _, p := range fallbackBinaryPaths(name) {
		if _, err := exec.LookPath(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("%s not found in PATH or common locations", name)
}

func fallbackBinaryPaths(name string) []string {
	return []string{
		"/usr/bin/" + name,
		"/usr/local/bin/" + name,
		"/opt/homebrew/bin/" + name,
		"/usr/local/opt/ffmpeg/bin/" + name,
		"/opt/homebrew/opt/ffmpeg/bin/" + name,
		"/usr/local/opt/ffmpeg-full/bin/" + name,
		"/opt/homebrew/opt/ffmpeg-full/bin/" + name,
	}
}

var versionRe = regexp.MustCompile(`ffmpeg version (\S+)`)

func parseVersion(ffmpegPath string) (version, buildConfig string, err error) {
	out, err := exec.Command(ffmpegPath, "-version").Output()
	if err != nil {
		return "", "", err
	}

	output := string(out)
	lines := strings.Split(output, "\n")

	matches := versionRe.FindStringSubmatch(output)
	if len(matches) >= 2 {
		version = matches[1]
	} else {
		version = "unknown"
	}

	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "configuration:") {
			buildConfig = strings.TrimPrefix(strings.TrimSpace(line), "configuration: ")
			break
		}
	}

	return version, buildConfig, nil
}

func findNextToExecutable(name string) (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("could not determine executable path: %w", err)
	}

	if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
		exePath = resolved
	}

	candidate := filepath.Join(filepath.Dir(exePath), name)
	if runtime.GOOS == "windows" {
		candidate += ".exe"
	}

	if fileExists(candidate) {
		return candidate, nil
	}

	return "", fmt.Errorf("%s not found next to executable (tried %s)", name, candidate)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
