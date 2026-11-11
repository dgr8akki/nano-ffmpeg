package ffmpeg

import "strings"

// errorTranslations maps ffmpeg error patterns to friendly messages.
var errorTranslations = []struct {
	pattern string
	message string
}{
	{"Unknown encoder", "The selected encoder is not available in your ffmpeg build. Try a different codec."},
	{"Encoder not found", "The selected encoder is not available in your ffmpeg build. Try a different codec."},
	{"No such file or directory", "The input file was not found. Check the file path."},
	{"Invalid data found when processing input", "The input file appears to be corrupted or in an unsupported format."},
	{"Output file is empty", "Encoding produced no output. Check your settings."},
	{"Permission denied", "Cannot write to the output directory. Check file permissions."},
	{"already exists. Overwrite", "Output file already exists and overwrite was not confirmed."},
	{"supported for WebM", "WebM only accepts VP9/AV1 video and Opus/Vorbis audio. Pick a different codec or format."},
	{"only supported in", "The selected codec is not compatible with the output format. Try a different codec or format."},
	{"-to value smaller than -ss", "End time must be after start time."},
	{"Invalid duration", "A time field is not a valid time. Use HH:MM:SS or seconds."},
	{"codec not currently supported in container", "The selected codec is not compatible with the output format. Try a different codec or format."},
	{"Could not find tag for codec", "The selected codec is not supported in this container format."},
	{"height not divisible by 2", "The output resolution must have even width and height. Try a different resolution."},
	{"width not divisible by 2", "The output resolution must have even width and height. Try a different resolution."},
	{"does not contain any stream", "The input file does not contain the expected stream type (video/audio)."},
	{"Avi timestamp discrepancy", "AVI format has timestamp issues. Try a different output format like MP4 or MKV."},
	{"Too many packets buffered", "Encoding buffer overflow. Try reducing quality or using a faster preset."},
	{"bitrate tolerance", "The target bitrate is too low for the selected resolution and codec."},
	{"No NVENC capable devices found", "NVIDIA hardware encoding is not available. Using software encoding instead."},
	{"Cannot load nvcuda.dll", "NVIDIA drivers not found. Hardware encoding is not available."},
	{"videotoolbox", "VideoToolbox hardware encoding failed. Try software encoding."},
	{"filter", "A video/audio filter failed. Check filter settings."},
	{"Discarding ID3 tags", ""}, // Suppress this common non-error
}

// TranslateError converts an ffmpeg error to a human-readable message.
func TranslateError(rawError string) string {
	for _, t := range errorTranslations {
		if strings.Contains(strings.ToLower(rawError), strings.ToLower(t.pattern)) {
			if t.message == "" {
				return "" // suppress non-errors
			}
			return t.message
		}
	}
	return rawError
}
