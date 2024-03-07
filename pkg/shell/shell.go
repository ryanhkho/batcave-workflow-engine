package shell

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os/exec"
)

const ExitOK int = 0
const ExitUnknown int = 232
const ExitContextCancel int = 231
const ExitKillFailure int = 230

// Command is any function that accepts optionFuncs and returns an exit code
//
// Most commands can take advantage of the run function which automatically
// parses the options to configure the exec.Cmd
//
// It also handles early termination of the command with a context and logging
type Command func(...OptionFunc) int

// Options are flexible parameters for any command
type Options struct {
	dryRunEnabled bool
	stdin         io.Reader
	stdout        io.Writer
	stderr        io.Writer
	ctx           context.Context
}

// apply should be called before the exec.Cmd is run
func (o *Options) apply(options ...OptionFunc) {
	for _, optionFunc := range options {
		optionFunc(o)
	}
}

// newOptions is used to generate an Options struct and automatically apply optionFuncs
func newOptions(options ...OptionFunc) *Options {
	o := new(Options)
	o.apply(options...)
	return o
}

// OptionFunc are used to set option values in a flexible way
type OptionFunc func(o *Options)

// WithDryRun sets the dryRunEnabled option which will print the command that would run and exitOK
func WithDryRun(enabled bool) OptionFunc {
	return func(o *Options) {
		o.dryRunEnabled = enabled
	}
}

// WithIO sets input and output for a command
func WithIO(stdin io.Reader, stdout io.Writer, stderr io.Writer) OptionFunc {
	return func(o *Options) {
		o.stdin = stdin
		o.stdout = stdout
		o.stderr = stderr
	}
}

// WithStdin only sets STDIN for the command
func WithStdin(r io.Reader) OptionFunc {
	return func(o *Options) {
		o.stdin = r
	}
}

// WithStdout only sets STDOUT for the command
func WithStdout(w io.Writer) OptionFunc {
	return func(o *Options) {
		o.stdout = w
	}
}

// WithStderr only sets STDERR for the command
func WithStderr(w io.Writer) OptionFunc {
	return func(o *Options) {
		o.stderr = w
	}
}

// run handles the execution of the command
//
// context will be set to background if not provided in the o.ctx
// this enables the command to be terminated before completion
// if ctx fires done.
//
// Setting the dry run option will always return ExitOK
func run(cmd *exec.Cmd, o *Options) int {
	slog.Info("shell exec", "dry_run", o.dryRunEnabled, "command", cmd.String())
	if o.dryRunEnabled {
		return ExitOK
	}

	cmd.Stdin = o.stdin
	cmd.Stdout = o.stdout
	cmd.Stderr = o.stderr

	if err := cmd.Start(); err != nil {
		return ExitUnknown
	}
	if o.ctx == nil {
		o.ctx = context.Background()
	}
	var runError error
	doneChan := make(chan struct{}, 1)
	go func() {
		runError = cmd.Wait()
		doneChan <- struct{}{}
	}()

	select {
	case <-o.ctx.Done():
		if err := cmd.Process.Kill(); err != nil {
			return ExitKillFailure
		}
		return ExitContextCancel
	case <-doneChan:
		var exitCodeError *exec.ExitError
		if errors.As(runError, &exitCodeError) {
			return exitCodeError.ExitCode()
		}
		if runError != nil {
			return ExitUnknown
		}
	}

	return ExitOK
}