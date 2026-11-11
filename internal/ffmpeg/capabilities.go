package ffmpeg

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Capabilities holds the parsed ffmpeg capabilities.
type Capabilities struct {
	Codecs    []Codec   `json:"codecs"`
	Formats   []Format  `json:"formats"`
	Filters   []string  `json:"filters"`
	HWAccels  []string  `json:"hwaccels"`
	Version   string    `json:"version"`
}

// Codec represents a supported codec.
type Codec struct {
	Name       string `json:"name"`
	Decoding   bool   `json:"decoding"`
	Encoding   bool   `json:"encoding"`
	Type       string `json:"type"` // video, audio, subtitle, data
	LossyComp  bool   `json:"lossy"`
	LosslessComp bool `json:"lossless"`
}

// Format represents a supported container format.
type Format struct {
	Name    string `json:"name"`
	Demux   bool   `json:"demux"`
	Mux     bool   `json:"mux"`
}

// ProbeCapabilities runs ffmpeg queries and returns all capabilities.
func ProbeCapabilities(info *Info) (*Capabilities, error) {
	// Check cache first
	cached, err := loadCachedCapabilities(info.Version)
	if err == nil && cached != nil {
		return cached, nil
	}

	caps := &Capabilities{Version: info.Version}

	codecs, err := parseCodecs(info.FFmpegPath)
	if err == nil {
		caps.Codecs = codecs
	}

	formats, err := parseFormats(info.FFmpegPath)
	if err == nil {
		caps.Formats = formats
	}

	filters, err := parseFilters(info.FFmpegPath)
	if err == nil {
		caps.Filters = filters
	}

	hwaccels, err := parseHWAccels(info.FFmpegPath)
	if err == nil {
		caps.HWAccels = hwaccels
	}

	_ = cacheCapabilities(caps)
	return caps, nil
}

// HasEncoder checks if a specific encoder is available.
func (c *Capabilities) HasEncoder(name string) bool {
	for _, codec := range c.Codecs {
		if codec.Name == name && codec.Encoding {
			return true
		}
	}
	return false
}

// HasFilter checks if a specific filter is available.
func (c *Capabilities) HasFilter(name string) bool {
	for _, f := range c.Filters {
		if f == name {
			return true
		}
	}
	return false
}

// HasHWAccel checks if a specific hardware accelerator is available.
func (c *Capabilities) HasHWAccel(name string) bool {
	for _, h := range c.HWAccels {
		if h == name {
			return true
		}
	}
	return false
}

var codecRe = regexp.MustCompile(`^\s*([D.])([E.])([VASDT])([I.])([L.])([S.])\s+(\S+)\s+`)

func parseCodecs(ffmpegPath string) ([]Codec, error) {
	out, err := exec.Command(ffmpegPath, "-codecs", "-hide_banner").Output()
	if err != nil {
		return nil, err
	}

	var codecs []Codec
	for _, line := range strings.Split(string(out), "\n") {
		matches := codecRe.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		codecType := "unknown"
		switch matches[3] {
		case "V":
			codecType = "video"
		case "A":
			codecType = "audio"
		case "S":
			codecType = "subtitle"
		case "D":
			codecType = "data"
		}

		codecs = append(codecs, Codec{
			Name:         matches[7],
			Decoding:     matches[1] == "D",
			Encoding:     matches[2] == "E",
			Type:         codecType,
			LossyComp:    matches[5] == "L",
			LosslessComp: matches[6] == "S",
		})
	}
	return codecs, nil
}

var formatRe = regexp.MustCompile(`^\s*([D ])([E ])[\s.]+(\S+)\s+`)

func parseFormats(ffmpegPath string) ([]Format, error) {
	out, err := exec.Command(ffmpegPath, "-formats", "-hide_banner").Output()
	if err != nil {
		return nil, err
	}

	var formats []Format
	inList := false
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "---") {
			inList = true
			continue
		}
		if !inList {
			continue
		}

		matches := formatRe.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		formats = append(formats, Format{
			Name:  matches[3],
			Demux: matches[1] == "D",
			Mux:   matches[2] == "E",
		})
	}
	return formats, nil
}

func parseFilters(ffmpegPath string) ([]string, error) {
	out, err := exec.Command(ffmpegPath, "-filters", "-hide_banner").Output()
	if err != nil {
		return nil, err
	}

	// ffmpeg <= 7 prints three flag columns (T.C), ffmpeg 8+ prints two (TS).
	filterNameRe := regexp.MustCompile(`^\s*[T.][S.][C.]?\s+(\S+)\s+\S*->`)
	var filters []string
	inList := false
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "------") {
			inList = true
			continue
		}
		if !inList {
			continue
		}

		matches := filterNameRe.FindStringSubmatch(line)
		if matches != nil {
			filters = append(filters, matches[1])
		}
	}
	return filters, nil
}

func parseHWAccels(ffmpegPath string) ([]string, error) {
	out, err := exec.Command(ffmpegPath, "-hwaccels", "-hide_banner").Output()
	if err != nil {
		return nil, err
	}

	var accels []string
	inList := false
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "Hardware acceleration methods:" {
			inList = true
			continue
		}
		if inList && line != "" {
			accels = append(accels, line)
		}
	}
	return accels, nil
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "nano-ffmpeg")
}

func cachePath() string {
	return filepath.Join(configDir(), "capabilities.json")
}

func loadCachedCapabilities(currentVersion string) (*Capabilities, error) {
	data, err := os.ReadFile(cachePath())
	if err != nil {
		return nil, err
	}

	var caps Capabilities
	if err := json.Unmarshal(data, &caps); err != nil {
		return nil, err
	}

	// An empty filter list is a cache written by the pre-ffmpeg-8 parser.
	if caps.Version != currentVersion || len(caps.Filters) == 0 {
		return nil, nil
	}

	return &caps, nil
}

func cacheCapabilities(caps *Capabilities) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(caps, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath(), data, 0644)
}
