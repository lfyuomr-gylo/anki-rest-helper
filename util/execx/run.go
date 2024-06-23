package execx

import (
	"anki-rest-enhancer/util/iox"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type Params struct {
	Command string
	Args    []string
	Stdin   string
	// Env is passed as is to exec.Cmd
	Env []string
}

// RunAndCollectOutput properly handles the case described in https://github.com/golang/go/issues/23019 , i.e.
// it doesn't hang if executed command spawns a long-living subprocess, passed its stdout to it and then exited shortly.
func RunAndCollectOutput(ctx context.Context, params Params) ([]byte, error) {
	cmd := exec.CommandContext(ctx, params.Command, params.Args...)
	if params.Stdin != "" {
		cmd.Stdin = strings.NewReader(params.Stdin)
	}
	cmd.Env = params.Env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	defer iox.Close(stdout)
	stdoutDone := readInBackground(stdout)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	defer func() { _ = stdout.Close() }()
	stderrDone := readInBackground(stderr)

	if err := cmd.Run(); err != nil {
		select {
		// NOTE: we can't read from the channel synchronously, as the stderr pipe
		// could have leaked to child processes spawned by the executed process,
		// which may be alive indefinitely long, keeping the reading goroutine hanging forever.
		case res := <-stderrDone:
			// we ignore any errors occurred while reading from the stderr as we're only interested in anything that
			// was read from there
			err = fmt.Errorf("%w\nScript stderr:\n%s", err, string(res.bytes))
		default:
			// continue execution
		}
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-stdoutDone:
		return res.bytes, res.err
	}
}

type readerResult struct {
	bytes []byte
	err   error
}

func readInBackground(r io.Reader) <-chan readerResult {
	done := make(chan readerResult, 1)
	go func() {
		bytes, err := io.ReadAll(r)
		done <- readerResult{bytes, err}
	}()
	return done
}
