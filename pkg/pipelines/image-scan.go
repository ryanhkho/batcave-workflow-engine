package pipelines

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"path"
	"workflow-engine/pkg/shell"
)

const mockSBOMFilename = "../../test/ubuntu_latest_20240125.syft_sbom.json"

type ImageScan struct {
	Stdout         io.Writer
	Stderr         io.Writer
	logger         *slog.Logger
	DryRunEnabled  bool
	artifactConfig ArtifactConfig
}

func (p *ImageScan) WithArtifactConfig(config ArtifactConfig) *ImageScan {
	p.artifactConfig = config
	return p
}

func NewImageScan(stdout io.Writer, stderr io.Writer) *ImageScan {
	return &ImageScan{
		Stdout: stdout,
		Stderr: stderr,
		artifactConfig: ArtifactConfig{
			Directory:    os.TempDir(),
			SBOMFilename: "sbom.json",
		},
		DryRunEnabled: false,
		logger:        slog.Default().With("pipeline", "image_scan"),
	}
}

func (p *ImageScan) Run() error {
	p.logger = p.logger.With("dry_run_enabled", p.DryRunEnabled)
	p.logger = p.logger.With(
		"artifact_config.directory", p.artifactConfig.Directory,
		"artifact_config.sbom_filename", p.artifactConfig.SBOMFilename,
		"artifact_config.grype_filename", p.artifactConfig.GrypeFilename,
	)

	// TODO: need syft SBOM output filename, it'll have to be saved in the artifact directory
	sbomFilename := path.Join(p.artifactConfig.Directory, p.artifactConfig.SBOMFilename)
	p.logger.Debug("SIMULATED: create SBOM by copying", "dest", sbomFilename)
	sbomFile, err := os.Open(mockSBOMFilename)
	if err != nil {
		return err
	}

	f, err := os.Open(sbomFilename)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, sbomFile); err != nil {
		return err
	}

	// TODO: End where the Syft code will go

	// Holds the grype scan output TODO: multi writer to the artifact directory and gatecheck
	buf := new(bytes.Buffer)

	// Do a grype scan on the SBOM, fail if the command fails
	err = shell.GrypeCommand(buf, p.Stderr).ScanSBOM(p.artifactConfig.SBOMFilename).WithDryRun(p.DryRunEnabled).Run()
	if err != nil {
		return err
	}

	// Save the grype file to the artifact directory
	grypeFilename := path.Join(p.artifactConfig.Directory, p.artifactConfig.GrypeFilename)
	p.logger.Debug("open grype artifact", "dest", grypeFilename)
	grypeFile, err := os.OpenFile(grypeFilename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	p.logger.Debug("save grype artifact", "dest", grypeFilename)
	if _, err := io.Copy(grypeFile, buf); err != nil {
		return err
	}

	return nil
}