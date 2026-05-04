package ffmpeg

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Command builds an ffmpeg command from structured options.
type Command struct {
	FFmpegPath string
	Input      string
	Output     string
	Args       []string
	Overwrite  bool
	Cleanup    []string
}

// AddCleanup registers temp files to remove once the command has finished.
func (c *Command) AddCleanup(paths ...string) *Command {
	c.Cleanup = append(c.Cleanup, paths...)
	return c
}

// NewCommand creates a new ffmpeg command builder.
func NewCommand(ffmpegPath, input, output string) *Command {
	return &Command{
		FFmpegPath: ffmpegPath,
		Input:      input,
		Output:     output,
		Overwrite:  true,
	}
}

// AddArg adds a single argument.
func (c *Command) AddArg(arg string) *Command {
	c.Args = append(c.Args, arg)
	return c
}

// AddArgs adds a flag-value pair.
func (c *Command) AddArgs(flag, value string) *Command {
	c.Args = append(c.Args, flag, value)
	return c
}

// SetVideoCodec sets the video codec.
func (c *Command) SetVideoCodec(codec string) *Command {
	return c.AddArgs("-c:v", codec)
}

// SetAudioCodec sets the audio codec.
func (c *Command) SetAudioCodec(codec string) *Command {
	return c.AddArgs("-c:a", codec)
}

// SetCRF sets the constant rate factor for quality.
func (c *Command) SetCRF(crf int) *Command {
	return c.AddArgs("-crf", fmt.Sprintf("%d", crf))
}

// SetPreset sets the encoding preset (ultrafast to veryslow).
func (c *Command) SetPreset(preset string) *Command {
	return c.AddArgs("-preset", preset)
}

// svtAV1Presets maps the named UI presets to libsvtav1's integer preset scale
// (0 = slowest/best quality, 13 = fastest). libsvtav1 rejects string presets
// like "slow" or "medium", so they must be translated before being passed to
// ffmpeg.
var svtAV1Presets = map[string]string{
	"slow":      "4",
	"medium":    "6",
	"fast":      "9",
	"ultrafast": "12",
}

// SetPresetForCodec sets the encoding preset using the right format for the
// given codec. libsvtav1 only accepts integer presets (0-13) and will fail
// with "Invalid argument" for string presets, so named presets are mapped to
// their integer equivalents. For all other codecs, the preset is passed
// through unchanged.
func (c *Command) SetPresetForCodec(codec, preset string) *Command {
	if codec == "libsvtav1" {
		key := strings.ToLower(strings.TrimSpace(preset))
		if mapped, ok := svtAV1Presets[key]; ok {
			return c.AddArgs("-preset", mapped)
		}
		// Pass through values that already look like an integer preset so
		// callers can opt into the raw 0-13 scale if they want.
		if _, err := strconv.Atoi(key); err == nil {
			return c.AddArgs("-preset", key)
		}
		// Unknown value: fall back to a sensible default rather than emit an
		// argument that ffmpeg will reject.
		return c.AddArgs("-preset", svtAV1Presets["medium"])
	}
	return c.AddArgs("-preset", preset)
}

// SetBitrate sets the overall bitrate.
func (c *Command) SetBitrate(bitrate string) *Command {
	return c.AddArgs("-b:v", bitrate)
}

// SetAudioBitrate sets the audio bitrate.
func (c *Command) SetAudioBitrate(bitrate string) *Command {
	return c.AddArgs("-b:a", bitrate)
}

// SetResolution sets output resolution.
func (c *Command) SetResolution(width, height int) *Command {
	return c.AddArgs("-vf", fmt.Sprintf("scale=%d:%d", width, height))
}

// SetScaleHeight scales to a specific height, keeping aspect ratio.
func (c *Command) SetScaleHeight(height int) *Command {
	return c.AddArgs("-vf", fmt.Sprintf("scale=-2:%d", height))
}

// SetStartTime sets the start time for trimming.
func (c *Command) SetStartTime(t string) *Command {
	return c.AddArgs("-ss", t)
}

// SetEndTime sets the end time for trimming.
func (c *Command) SetEndTime(t string) *Command {
	return c.AddArgs("-to", t)
}

// SetDuration sets the duration.
func (c *Command) SetDuration(d string) *Command {
	return c.AddArgs("-t", d)
}

// StreamCopy copies streams without re-encoding.
func (c *Command) StreamCopy() *Command {
	return c.AddArgs("-c", "copy")
}

// NoVideo removes video stream.
func (c *Command) NoVideo() *Command {
	return c.AddArg("-vn")
}

// NoAudio removes audio stream.
func (c *Command) NoAudio() *Command {
	return c.AddArg("-an")
}

// AddVideoFilter adds a video filter.
func (c *Command) AddVideoFilter(filter string) *Command {
	return c.AddArgs("-vf", filter)
}

// AddAudioFilter adds an audio filter.
func (c *Command) AddAudioFilter(filter string) *Command {
	return c.AddArgs("-af", filter)
}

// SetFrameRate sets output frame rate.
func (c *Command) SetFrameRate(fps int) *Command {
	return c.AddArgs("-r", fmt.Sprintf("%d", fps))
}

// SetPixelFormat sets the pixel format.
func (c *Command) SetPixelFormat(fmt string) *Command {
	return c.AddArgs("-pix_fmt", fmt)
}

// SetHWAccel sets hardware acceleration.
func (c *Command) SetHWAccel(accel string) *Command {
	return c.AddArgs("-hwaccel", accel)
}

// SetVideoEncoder sets the video encoder with HW acceleration.
func (c *Command) SetVideoEncoder(encoder string) *Command {
	return c.AddArgs("-c:v", encoder)
}

// Build returns the full argument list.
func (c *Command) Build() []string {
	args := []string{}
	if c.Overwrite {
		args = append(args, "-y")
	}
	args = append(args, "-i", c.Input)
	args = append(args, c.Args...)
	args = append(args, c.Output)
	return args
}

// String returns the command as a human-readable string.
func (c *Command) String() string {
	args := c.Build()
	parts := []string{c.FFmpegPath}
	parts = append(parts, args...)

	// Quote args with spaces
	quoted := make([]string, len(parts))
	for i, p := range parts {
		if strings.Contains(p, " ") {
			quoted[i] = fmt.Sprintf("%q", p)
		} else {
			quoted[i] = p
		}
	}
	return strings.Join(quoted, " ")
}

// Exec creates an os/exec.Cmd ready to run.
func (c *Command) Exec() *exec.Cmd {
	return exec.Command(c.FFmpegPath, c.Build()...)
}
