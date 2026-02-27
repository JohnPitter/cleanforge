package cmd

import (
	"context"
	"os/exec"
	"syscall"
)

// Hidden creates an exec.Cmd with the CREATE_NO_WINDOW flag set,
// preventing a visible console window from appearing when running
// subprocesses from a GUI application.
func Hidden(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	return cmd
}

// HiddenContext creates a context-aware exec.Cmd with the CREATE_NO_WINDOW flag.
// The command will be killed when the context deadline is exceeded.
func HiddenContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	return cmd
}
