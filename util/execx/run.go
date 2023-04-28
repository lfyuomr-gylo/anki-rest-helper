package execx

import (
	"context"
	"io"
	"os/exec"
	"strings"
)

type Params struct {
	Command string
	Args    []string
	Stdin   string
}

// RunAndCollectOutput properly handles the case described in https://github.com/golang/go/issues/23019 , i.e.
// it doesn't hang if executed command spawns a long-living subprocess, passed its stdout to it and then exited shortly.
func RunAndCollectOutput(ctx context.Context, params Params) ([]byte, error) {
	cmd := exec.CommandContext(ctx, params.Command, params.Args...)
	if params.Stdin != "" {
		cmd.Stdin = strings.NewReader(params.Stdin)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	defer func() { _ = stdout.Close() }()

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
