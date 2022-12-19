package execx

import (
	"anki-rest-enhancer/util/iox"
	"context"
	_ "embed"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

//go:embed hang.sh
var hangScript []byte

func TestRunAndCollectOutput_Timeout(t *testing.T) {
	// setup:
	scriptFileName := writeIntoTmp(t, hangScript)
	defer func() { _ = os.Remove(scriptFileName) }()

	// when:
	const timeout = 500 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	start := time.Now()
	_, err := RunAndCollectOutput(ctx, Params{Command: "bash", Args: []string{scriptFileName}})
	duration := time.Now().Sub(start)
	require.Error(t, err)
	require.True(t, duration < 2*timeout)
}

func TestRunAndCollectOutput_Success(t *testing.T) {
	// when:
	output, err := RunAndCollectOutput(context.Background(), Params{
		Command: "echo",
		Args:    []string{"foo"},
	})

	// then:
	require.NoError(t, err)
	require.Equal(t, "foo\n", string(output))
}
func writeIntoTmp(t *testing.T, content []byte) (fileName string) {
	tmpFile, err := os.CreateTemp("", "")
	defer iox.Close(tmpFile)
	require.NoError(t, err)
	_, err = tmpFile.Write(content)
	require.NoError(t, err)
	return tmpFile.Name()
}
