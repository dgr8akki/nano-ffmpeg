package ffmpeg

import (
	"bufio"
	"io"
	"os"
	"os/exec"
)

// RunnerMsg types for Bubble Tea message passing.

// ProgressMsg carries a progress update.
type ProgressMsg struct {
	Progress *Progress
}

// LogMsg carries a raw stderr line.
type LogMsg struct {
	Line string
}

// DoneMsg signals ffmpeg has finished.
type DoneMsg struct {
	Err        error
	OutputPath string
}

// Runner manages an ffmpeg process.
type Runner struct {
	cmd        *exec.Cmd
	outputPath string
	stderr     io.ReadCloser
}

// NewRunner creates a runner for the given command.
func NewRunner(ffmpegCmd *Command) (*Runner, error) {
	cmd := ffmpegCmd.Exec()

	// Put the child in its own process group so Cancel can signal the whole
	// tree (ffmpeg, any shell wrappers, spawned children). No-op on Windows.
	configureProcessGroup(cmd)

	// Capture stderr (ffmpeg outputs progress there)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	// Suppress stdout
	cmd.Stdout = nil

	return &Runner{
		cmd:        cmd,
		outputPath: ffmpegCmd.Output,
		stderr:     stderr,
	}, nil
}

// Start begins the ffmpeg process.
func (r *Runner) Start() error {
	return r.cmd.Start()
}

// OutputPath returns the output file path.
func (r *Runner) OutputPath() string {
	return r.outputPath
}

// ScanStderr returns a scanner over the process stderr.
func (r *Runner) ScanStderr() *bufio.Scanner {
	scanner := bufio.NewScanner(r.stderr)
	// ffmpeg progress lines use \r, so split on either \n or \r
	scanner.Split(scanLinesOrCR)
	return scanner
}

// Wait waits for the process to exit.
func (r *Runner) Wait() error {
	return r.cmd.Wait()
}

// Cancel asks the running process (and its process group, on POSIX) to stop
// gracefully by delivering an interrupt signal. It is a no-op if Start has not
// yet been called or the process has already exited.
func (r *Runner) Cancel() error {
	if r.cmd == nil || r.cmd.Process == nil {
		return nil
	}
	return sendInterrupt(r.cmd)
}

// Kill force-stops the process if it ignored Cancel.
func (r *Runner) Kill() error {
	if r.cmd == nil || r.cmd.Process == nil {
		return nil
	}
	return r.cmd.Process.Kill()
}

// CleanupOutput removes the partial output file.
func (r *Runner) CleanupOutput() {
	if r.outputPath == "" || r.outputPath == "-" {
		return
	}
	os.Remove(r.outputPath)
}

// scanLinesOrCR splits input on \n or \r (ffmpeg uses \r for progress).
func scanLinesOrCR(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	for i, b := range data {
		if b == '\n' || b == '\r' {
			return i + 1, data[:i], nil
		}
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}
