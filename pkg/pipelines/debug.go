package pipelines

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"workflow-engine/pkg/shell"
)

type Debug struct {
	Stdout        io.Writer
	Stderr        io.Writer
	DryRunEnabled bool
}

// NewDebug creates a new debug pipeline with custom stdout and stderr
func NewDebug(stdoutW io.Writer, stderrW io.Writer) *Debug {
	return &Debug{Stdout: stdoutW, Stderr: stderrW, DryRunEnabled: false}
}

// Run prints the version for all expected commands
//
// All commands will run in sequence, stopping if one of the commands fail
func (d *Debug) Run() error {
	l := slog.Default().With("pipeline", "debug", "dry_run", d.DryRunEnabled)
	l.Info("start")

	config := NewDefaultConfig()
	config.Syft.ImageTarball = "./test/.artifacts/build-image/test-local.tar"
	config.Syft.ImageSbom = "./test/.artifacts/sbom/sbom.json"

	ld := slog.Default()

	// Get current directory
	pwd, err := os.Getwd()
	if err != nil {
		ld.Error(fmt.Sprintln(err))
		os.Exit(1)
	}
	ld.Info(fmt.Sprintf("Current directory: %s", pwd))

	// Create artifact directory
	artifactDir := filepath.Join(pwd, "test", ".artifacts", "build-image")
	err = os.MkdirAll(artifactDir, os.ModePerm)
	if err != nil {
		ld.Error(fmt.Sprintln(err))
		os.Exit(1)
	}

	// Collect errors for mandatory commands
	errs := errors.Join(
		shell.GrypeCommand(d.Stdout, d.Stderr).Version().WithDryRun(d.DryRunEnabled).Run(),
		shell.SyftCommand(d.Stdout, d.Stderr).Version().WithDryRun(d.DryRunEnabled).Run(),

		// TODO: Add docker build and save commands here

		shell.SyftCommand(d.Stdout, d.Stderr).ScanImage(config.Syft.ImageTarball, config.Syft.ImageSbom).WithDryRun(d.DryRunEnabled).Run(),
	)

	// Just log errors for optional commands
	shell.PodmanCommand(d.Stdout, d.Stderr).Version().WithDryRun(d.DryRunEnabled).RunLogErrorAsWarning()
	shell.DockerCommand(d.Stdout, d.Stderr).Version().WithDryRun(d.DryRunEnabled).RunLogErrorAsWarning()

	l.Info("complete")
	return errs
}
