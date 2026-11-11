package ffmpeg

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Progress holds parsed ffmpeg progress data from stderr.
type Progress struct {
	Frame     int64
	FPS       float64
	Quality   float64
	Size      int64   // bytes
	Time      float64 // seconds
	Bitrate   float64 // kbits/s
	Speed     float64 // e.g. 2.3 means 2.3x realtime
	Percent   float64 // 0-100, calculated from duration
	ETA       time.Duration
	Elapsed   time.Duration
	Pass      int
	TotalPass int
}

var (
	frameRe   = regexp.MustCompile(`frame=\s*(\d+)`)
	fpsRe     = regexp.MustCompile(`fps=\s*([\d.]+)`)
	qualityRe = regexp.MustCompile(`q=\s*([\d.-]+)`)
	sizeRe    = regexp.MustCompile(`L?size=\s*(\d+)\s*(?:kB|KiB)`) // ffmpeg 7+ prints KiB
	timeRe    = regexp.MustCompile(`time=\s*(\d+):(\d+):(\d+)\.(\d+)`)
	bitrateRe = regexp.MustCompile(`bitrate=\s*([\d.]+)\s*kbits/s`)
	speedRe   = regexp.MustCompile(`speed=\s*([\d.]+)x`)
)

// ProgressParser maintains state for smoothing ETA calculations.
type ProgressParser struct {
	TotalDuration float64
	StartTime     time.Time
	etaHistory    []float64
	maxHistory    int
}

// NewProgressParser creates a parser with the total duration for percentage calculation.
func NewProgressParser(totalDuration float64) *ProgressParser {
	return &ProgressParser{
		TotalDuration: totalDuration,
		StartTime:     time.Now(),
		maxHistory:    5,
	}
}

// Parse extracts progress data from an ffmpeg stderr line.
func (pp *ProgressParser) Parse(line string) *Progress {
	// Only parse lines that look like progress output
	if !strings.Contains(line, "frame=") && !strings.Contains(line, "size=") {
		return nil
	}

	p := &Progress{
		Elapsed: time.Since(pp.StartTime),
	}

	if m := frameRe.FindStringSubmatch(line); m != nil {
		p.Frame, _ = strconv.ParseInt(m[1], 10, 64)
	}

	if m := fpsRe.FindStringSubmatch(line); m != nil {
		p.FPS, _ = strconv.ParseFloat(m[1], 64)
	}

	if m := qualityRe.FindStringSubmatch(line); m != nil {
		p.Quality, _ = strconv.ParseFloat(m[1], 64)
	}

	if m := sizeRe.FindStringSubmatch(line); m != nil {
		kb, _ := strconv.ParseInt(m[1], 10, 64)
		p.Size = kb * 1024
	}

	if m := timeRe.FindStringSubmatch(line); m != nil {
		hours, _ := strconv.Atoi(m[1])
		mins, _ := strconv.Atoi(m[2])
		secs, _ := strconv.Atoi(m[3])
		ms, _ := strconv.Atoi(m[4])
		p.Time = float64(hours)*3600 + float64(mins)*60 + float64(secs) + float64(ms)/100
	}

	if m := bitrateRe.FindStringSubmatch(line); m != nil {
		p.Bitrate, _ = strconv.ParseFloat(m[1], 64)
	}

	if m := speedRe.FindStringSubmatch(line); m != nil {
		p.Speed, _ = strconv.ParseFloat(m[1], 64)
	}

	// Calculate percentage
	if pp.TotalDuration > 0 && p.Time > 0 {
		p.Percent = (p.Time / pp.TotalDuration) * 100
		if p.Percent > 100 {
			p.Percent = 100
		}
	}

	// Calculate ETA with smoothing
	if p.Speed > 0 && pp.TotalDuration > 0 {
		remaining := pp.TotalDuration - p.Time
		etaSeconds := remaining / p.Speed

		pp.etaHistory = append(pp.etaHistory, etaSeconds)
		if len(pp.etaHistory) > pp.maxHistory {
			pp.etaHistory = pp.etaHistory[1:]
		}

		// Rolling average for smooth ETA
		sum := 0.0
		for _, e := range pp.etaHistory {
			sum += e
		}
		avgETA := sum / float64(len(pp.etaHistory))
		p.ETA = time.Duration(avgETA * float64(time.Second))
	}

	return p
}

// FormatDuration formats a duration as HH:MM:SS.
func FormatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// FormatSize formats bytes to human-readable.
func FormatSize(bytes int64) string {
	size := float64(bytes)
	units := []string{"B", "KB", "MB", "GB"}
	i := 0
	for size >= 1024 && i < len(units)-1 {
		size /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%.0f %s", size, units[i])
	}
	return fmt.Sprintf("%.1f %s", size, units[i])
}
