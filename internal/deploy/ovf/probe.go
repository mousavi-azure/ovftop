package ovf

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// Probe runs `ovftool --machineOutput <source>` (no target, so ovftool
// just inspects the package) and returns a UI-friendly Descriptor.
// source may be a local .ovf/.ova path or a remote http(s)/ftp URL —
// ovftool handles all of these natively.
func Probe(ctx context.Context, ovftoolPath, source string) (*Descriptor, error) {
	cmd := exec.CommandContext(ctx, ovftoolPath, "--machineOutput", source)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	// Probe failures are reported through ovftool's own exit code and the
	// ERROR section, not a Go-level error we need to distinguish, so the
	// run error (if any) is intentionally not returned directly.
	_ = cmd.Run()

	out, err := ParseMachineOutput(stdout.String())
	if err != nil {
		return nil, fmt.Errorf("parsing ovftool output: %w", err)
	}
	if out.Result != "SUCCESS" {
		if len(out.Errors) > 0 {
			return nil, fmt.Errorf("%s", out.Errors[0])
		}
		return nil, fmt.Errorf("ovftool probe failed: %s", stdout.String())
	}
	if out.Probe == nil {
		return nil, fmt.Errorf("ovftool probe returned no data")
	}

	d := toDescriptor(out.Probe)
	d.Warnings = out.Warnings
	return d, nil
}
