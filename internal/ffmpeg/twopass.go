package ffmpeg

import (
	"fmt"
	"os"
	"path/filepath"
)

var twoPassCodecs = map[string]bool{
	"libx264":    true,
	"libx265":    true,
	"libvpx-vp9": true,
}

// TwoPassSupported reports whether the given video encoder can be driven with
// ffmpeg's -pass 1/-pass 2 workflow.
func TwoPassSupported(codec string) bool {
	return twoPassCodecs[codec]
}

// TwoPassStatsPrefix returns a unique stats-file prefix in the OS temp dir.
func TwoPassStatsPrefix() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("nano-ffmpeg-2pass-%d", os.Getpid()))
}

// BuildTwoPassCommands returns the pass-1 (analysis) and pass-2 (encode)
// ffmpeg commands for the given encoder, preset and target bitrate. The
// stats file written by pass 1 is consumed by pass 2; callers should remove
// the files matching prefix* once both passes finish.
func BuildTwoPassCommands(ffmpegPath, input, output, codec, preset, bitrate, statsPrefix string) []*Command {
	pass1 := NewCommand(ffmpegPath, input, os.DevNull)
	pass1.SetVideoCodec(codec)
	pass1.SetBitrate(bitrate)
	pass1.SetPresetForCodec(codec, preset)
	pass1.AddArgs("-pass", "1")
	pass1.AddArgs("-passlogfile", statsPrefix)
	pass1.NoAudio()
	pass1.AddArgs("-f", "null")
	pass1.AddCleanup(twoPassStatsArtifacts(statsPrefix)...)

	pass2 := NewCommand(ffmpegPath, input, output)
	pass2.SetVideoCodec(codec)
	pass2.SetBitrate(bitrate)
	pass2.SetPresetForCodec(codec, preset)
	pass2.AddArgs("-pass", "2")
	pass2.AddArgs("-passlogfile", statsPrefix)
	pass2.SetAudioCodec("copy")
	pass2.AddCleanup(twoPassStatsArtifacts(statsPrefix)...)

	return []*Command{pass1, pass2}
}

func twoPassStatsArtifacts(prefix string) []string {
	return []string{
		prefix + "-0.log",
		prefix + "-0.log.mbtree",
		prefix + ".log",
		prefix + ".log.mbtree",
	}
}
