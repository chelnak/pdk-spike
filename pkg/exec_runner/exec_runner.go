// Package exec_runner contains utility functions for executing commands.
package exec_runner // nolint

import (
	"os"
	"os/exec"
	"runtime"
)

type ExecRunner interface {
	Command(name string, arg ...string) error
	Output() ([]byte, error)
}

type execRunner struct {
	cmd *exec.Cmd
}

func (e *execRunner) Command(name string, args ...string) error {
	var pathToExecutable string
	var err error

	if runtime.GOOS == "windows" {
		pathToExecutable, err = exec.LookPath("cmd.exe")
	} else {
		pathToExecutable, err = exec.LookPath(name)
	}

	if err != nil {
		return err
	}

	cmd := &exec.Cmd{
		Path: pathToExecutable,
		Args: buildCommandArgs(name, args),
		Env:  os.Environ(),
	}
	e.cmd = cmd
	return nil
}

func (e *execRunner) Output() ([]byte, error) {
	return e.cmd.Output()
}

func buildCommandArgs(commandName string, args []string) []string {
	var cmd []string

	if runtime.GOOS == "windows" {
		cmd = append(cmd, "/c")
	}
	cmd = append(cmd, commandName)

	return append(cmd, args...)
}

func NewExecRunner() ExecRunner {
	return &execRunner{}
}
