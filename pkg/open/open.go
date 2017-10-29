package open

import (
	"os/exec"
	"runtime"
)

// Open open a file or other things with OS-specific default program.
// Ref: https://stackoverflow.com/a/39324149/2996656
func Open(args ...string) error {
	var cmd string
	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	return exec.Command(cmd, args...).Start()
}
