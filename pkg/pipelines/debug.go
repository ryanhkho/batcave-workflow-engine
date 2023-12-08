package pipelines

import (
	"fmt"
	"io"
	"strings"
	"workflow-engine/pkg/environments"
	"workflow-engine/pkg/jobs"

	"dagger.io/dagger"
	"golang.org/x/sync/errgroup"
)

// Debug pipeline is designed for smoke testing features
type Debug struct {
	name   string
	client *dagger.Client
	stdout io.Writer
}

// NewDebugPipeline setups an instance of the debug pipeline
func NewDebugPipeline(c *dagger.Client, stdoutWriter io.Writer) *Debug {
	pipeline := &Debug{name: "Debug Pipeline", client: c, stdout: stdoutWriter}
	return pipeline
}

// Execute runs the full debug pipeline
func (p *Debug) Execute() error {
	var errGroup errgroup.Group
	alpine := environments.NewAlpine(p.client)
	var debugOutput, systemInfo string
	// Get sample output in a go routine so it runs concurrently with the other tasks
	errGroup.Go(func() error {
		var err error
		debugOutput, err = jobs.RunDebug(alpine.Container())
		return err
	})

	// Get the debug system information
	errGroup.Go(func() error {
		var err error
		systemInfo, err = jobs.RunDebugSysInfo(alpine.Container())
		return err
	})

	// Caller handles any errors
	err := errGroup.Wait()
	r := strings.NewReader(fmt.Sprintf("debug output:\n%s\nsystem information:\n%s\n", debugOutput, systemInfo))
	r.WriteTo(p.stdout)
	return err
}
