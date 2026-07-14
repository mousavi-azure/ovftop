package ovf

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// section is one named block of ovftool's --machineOutput stream, e.g.
// "PROBE", "WARNING", "ERROR", "RESULT". Blocks are separated by a blank
// line; every body line is prefixed with "+ " (or a bare "+" for empty
// lines), which — once stripped — reassembles into well-formed XML for
// PROBE/WARNING/ERROR, or a bare literal token for RESULT.
type section struct {
	name string
	body string
}

func splitSections(raw string) []section {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	blocks := strings.Split(raw, "\n\n")

	var sections []section
	for _, block := range blocks {
		lines := strings.Split(strings.Trim(block, "\n"), "\n")
		if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
			continue
		}
		name := strings.TrimSpace(lines[0])

		var bodyLines []string
		for _, l := range lines[1:] {
			switch {
			case strings.HasPrefix(l, "+ "):
				bodyLines = append(bodyLines, l[2:])
			case l == "+":
				bodyLines = append(bodyLines, "")
			}
		}
		// Each stripped line is already a complete open tag, close tag, or
		// leaf text value (that's how ovftool emits --machineOutput), so
		// concatenating with no separator reconstructs valid XML without
		// injecting stray newlines into element text content.
		sections = append(sections, section{name: name, body: strings.Join(bodyLines, "")})
	}
	return sections
}

// ProbeOutput is the parsed result of an `ovftool --machineOutput <source>`
// probe run.
type ProbeOutput struct {
	Result   string
	Probe    *probeResult
	Warnings []string
	Errors   []string
}

// ParseMachineOutput parses the combined stdout+stderr of an ovftool probe
// invocation run with --machineOutput.
func ParseMachineOutput(raw string) (*ProbeOutput, error) {
	out := &ProbeOutput{}
	for _, s := range splitSections(raw) {
		switch s.name {
		case "PROBE":
			var pr probeResult
			if err := xml.Unmarshal([]byte(s.body), &pr); err != nil {
				return nil, fmt.Errorf("parsing PROBE section: %w", err)
			}
			out.Probe = &pr
		case "WARNING":
			var w probeWarnings
			if err := xml.Unmarshal([]byte(s.body), &w); err != nil {
				return nil, fmt.Errorf("parsing WARNING section: %w", err)
			}
			for _, item := range w.Warning {
				out.Warnings = append(out.Warnings, strings.TrimSpace(item.LocalizedMsg))
			}
		case "ERROR":
			var e probeErrors
			if err := xml.Unmarshal([]byte(s.body), &e); err != nil {
				return nil, fmt.Errorf("parsing ERROR section: %w", err)
			}
			for _, item := range e.Error {
				out.Errors = append(out.Errors, strings.TrimSpace(item.LocalizedMsg))
			}
		case "RESULT":
			out.Result = strings.TrimSpace(s.body)
		}
	}
	return out, nil
}
