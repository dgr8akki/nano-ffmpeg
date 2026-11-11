//go:build windows

package ffmpeg

import (
	"os/exec"
)

// configureProcessGroup is a no-op on Windows. Windows does not have POSIX
// process groups; signalling the process directly is the best we can do.
func configureProcessGroup(_ *exec.Cmd) {}

// sendInterrupt stops the process. Go does not implement os.Interrupt on
// Windows (Signal returns an error), so the process is killed outright.
func sendInterrupt(cmd *exec.Cmd) error {
	return cmd.Process.Kill()
}
