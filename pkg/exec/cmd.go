package exec

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
)

type Cmd struct{}

type CmdArgs struct {
	Dir  string
	Run  string
	Args []string
}

func (c *Cmd) ExecuteWithStream(opts CmdArgs) error {
	cmd := exec.Command(opts.Run, opts.Args...)
	cmd.Dir = opts.Dir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	scanner := bufio.NewScanner(stdout)
	scannerErr := bufio.NewScanner(stderr)
	for scanner.Scan() {
		fmt.Fprintln(os.Stdout, scanner.Text())
	}

	for scannerErr.Scan() {
		fmt.Fprintln(os.Stderr, scannerErr.Text())
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if err := scannerErr.Err(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}
