// Package exec_runner contains utility functions for executing commands.
package exec_runner // nolint

import (
	"os"
	"os/exec"
	"runtime"
)

type ExecI interface {
	Command(name string, arg ...string) error
	Output() ([]byte, error)
}

type Exec struct {
	cmd *exec.Cmd
}

func (e *Exec) Command(name string, arg ...string) error {
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

	correctArgs := buildCommandArgs(name, arg)
	cmd := &exec.Cmd{
		Path: pathToExecutable,
		Args: correctArgs,
		Env:  os.Environ(),
	}
	e.cmd = cmd
	return nil
}

func (e *Exec) Output() ([]byte, error) {
	return e.cmd.Output()
}

func buildCommandArgs(commandName string, args []string) []string {
	var a []string
	if runtime.GOOS == "windows" {
		a = append(a, "/c")
	}
	a = append(a, commandName)
	a = append(a, args...)
	return a
}
