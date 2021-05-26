package main

import (
	"os"
	"os/exec"
)

func main() {
	var cmd *exec.Cmd
	var err error

	bin, args := os.Args[1], os.Args[2:]

	cmd = exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err = cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
			return
		}

		os.Exit(1)
		return
	}

	os.Exit(0)
}
