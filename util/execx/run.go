package execx

import (
	"context"
	"io"
	"os/exec"
)

// RunAndCollectOutput properly handles the case described in https://github.com/golang/go/issues/23019 , i.e.
// it doesn't hang if executed command spawns a long-living subprocess, passed its stdout to it and then exited shortly.
func RunAndCollectOutput(ctx context.Context, command string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	stdout, err := cmd.StdoutPipe()
	defer func() { _ = stdout.Close() }()
	if err != nil {
		return nil, err
	}

	type readerResult struct {
		bytes []byte
		err   error
	}
	readerDone := make(chan readerResult, 1)
	go func() {
		bytes, err := io.ReadAll(stdout)
		readerDone <- readerResult{bytes, err}
	}()

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-readerDone:
		return res.bytes, res.err
	}
}
