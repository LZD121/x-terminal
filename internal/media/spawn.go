package media

import (
	"context"
	"os/exec"
	"strings"
)

// DefaultLookPath is exec.LookPath, exposed so the composition root can wire
// External without importing os/exec itself. The module-wide boundary holds
// os/exec to internal/term and this package.
var DefaultLookPath = exec.LookPath

// StartAndReap starts name with args and returns as soon as the process has
// been started. The returned channel delivers Wait's error (nil on success)
// exactly once; a goroutine reaps the child so no zombie accumulates and the
// pane's frame stays responsive. A caller that must surface a non-zero exit
// (MEDIA-03) waits on the channel from a tea.Cmd, never inside Open.
// Nothing here reads the child's output: a program's stdout is not this
// program's data.
//
// The argv is never joined into a shell command line: name is the program and
// args is the argument vector, passed to CommandContext as-is.
func StartAndReap(ctx context.Context, name string, args []string) (<-chan error, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	ch := make(chan error, 1)
	go func() {
		ch <- cmd.Wait()
		close(ch)
	}()
	return ch, nil
}

// RunWithStdin starts name with args, writes stdin to the child's stdin, waits
// for exit, and returns any start/wait/write error. Clipboard uses it for
// wl-copy: the URL must land on stdin, not argv. The argv is never joined into
// a shell command line.
func RunWithStdin(ctx context.Context, name string, args []string, stdin string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = strings.NewReader(stdin)
	return cmd.Run()
}
