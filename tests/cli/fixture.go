package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Fixture struct {
	tempDir string
	binPath string
}

const (
	clientBuildTarget = "./cmd/hours"
	serverBuildTarget = "./cmd/hours-server"
)

type HoursCmd struct {
	args  []string
	useDB bool
	env   map[string]string
}

func NewCmd(args []string) HoursCmd {
	return HoursCmd{
		args: args,
		env:  make(map[string]string),
	}
}

func (c *HoursCmd) AddArgs(args ...string) {
	c.args = append(c.args, args...)
}

func (c *HoursCmd) SetEnv(key, value string) {
	c.env[key] = value
}

func (c *HoursCmd) UseDB() {
	c.useDB = true
}

func NewFixture() (Fixture, error) {
	return buildFixture("hours", clientBuildTarget)
}

func NewServerFixture() (Fixture, error) {
	return buildFixture("hours-server", serverBuildTarget)
}

func buildFixture(binaryName string, buildTarget string) (Fixture, error) {
	var zero Fixture
	repoRoot, err := repoRootDir()
	if err != nil {
		return zero, err
	}

	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		return zero, fmt.Errorf("couldn't create temporary directory: %s", err.Error())
	}

	binPath := filepath.Join(tempDir, binaryName)
	buildArgs := []string{"build", "-o", binPath, buildTarget}

	buildCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	c := exec.CommandContext(buildCtx, "go", buildArgs...)
	c.Dir = repoRoot
	buildOutput, err := c.CombinedOutput()
	if err != nil {
		cleanupErr := os.RemoveAll(tempDir)
		if cleanupErr != nil {
			fmt.Fprintf(os.Stderr, "couldn't clean up temporary directory (%s): %s", tempDir, cleanupErr.Error())
		}

		return zero, fmt.Errorf(`couldn't build binary: %s
output:
%s`, err.Error(), buildOutput)
	}

	return Fixture{
		tempDir: tempDir,
		binPath: binPath,
	}, nil
}

func repoRootDir() (string, error) {
	gomodCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c := exec.CommandContext(gomodCtx, "go", "env", "GOMOD")
	gomodOutput, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("couldn't resolve repository root via go env GOMOD: %w\noutput:\n%s", err, gomodOutput)
	}

	gomodPath := filepath.Clean(strings.TrimSpace(string(gomodOutput)))
	if gomodPath == "" || gomodPath == os.DevNull {
		return "", errors.New("couldn't resolve repository root via go env GOMOD")
	}

	repoRoot := filepath.Dir(gomodPath)
	if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err != nil {
		return "", fmt.Errorf("couldn't resolve repository root from fixture helper: %w", err)
	}

	return repoRoot, nil
}

func (f Fixture) Cleanup() error {
	err := os.RemoveAll(f.tempDir)
	if err != nil {
		return fmt.Errorf("couldn't clean up temporary directory (%s): %s", f.tempDir, err.Error())
	}

	return nil
}

func (f Fixture) TempDir() string {
	return f.tempDir
}

func (f Fixture) BinPath() string {
	return f.binPath
}

func (f Fixture) RunCmd(cmd HoursCmd) (string, error) {
	argsToUse := cmd.args
	if cmd.useDB {
		dbPath := filepath.Join(f.tempDir, "hours.db")
		argsToUse = append(argsToUse, "--dbpath", dbPath)
	}
	runCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmdToRun := exec.CommandContext(runCtx, f.binPath, argsToUse...)

	cmdToRun.Env = os.Environ()
	for key, value := range cmd.env {
		cmdToRun.Env = append(cmdToRun.Env, fmt.Sprintf("%s=%s", key, value))
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmdToRun.Stdout = &stdoutBuf
	cmdToRun.Stderr = &stderrBuf

	err := cmdToRun.Run()
	exitCode := 0
	success := true

	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			success = false
			exitCode = exitError.ExitCode()
		} else {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(runCtx.Err(), context.DeadlineExceeded) {
				return "", fmt.Errorf(`command timed out after 30s: %w
----- stdout -----
%s
----- stderr -----
%s`, err, stdoutBuf.String(), stderrBuf.String())
			}

			return "", fmt.Errorf(`couldn't run command: %w
----- stdout -----
%s
----- stderr -----
%s`, err, stdoutBuf.String(), stderrBuf.String())
		}
	}

	output := fmt.Sprintf(`success: %t
exit_code: %d
----- stdout -----
%s
----- stderr -----
%s
`, success, exitCode, stdoutBuf.String(), stderrBuf.String())

	return output, nil
}
