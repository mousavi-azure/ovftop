package deploy

import (
	"fmt"
	"sort"
	"strings"
)

// BuildArgs turns a Spec into the argv ovftool should be invoked with. It
// is a pure function so the exact command can be previewed/edited in the
// wizard's summary step before anything runs.
func BuildArgs(spec Spec) []string {
	args := []string{
		"--noSSLVerify",
		"--acceptAllEulas",
		"--overwrite",
		"--I:morefArgs",
	}

	if spec.VMName != "" {
		args = append(args, "-n="+spec.VMName)
	}
	if spec.DiskMode != "" {
		args = append(args, "-dm="+string(spec.DiskMode))
	}
	if spec.DatastoreRef.Value != "" {
		args = append(args, "-ds="+MorefArg(spec.DatastoreRef))
	}
	if spec.FolderRef != nil {
		args = append(args, "--vmFolder="+MorefArg(*spec.FolderRef))
	}
	if spec.PowerOn {
		args = append(args, "--powerOn")
	}
	if spec.ImportAsTemplate {
		args = append(args, "--importAsTemplate")
	}

	for _, n := range spec.Networks {
		args = append(args, fmt.Sprintf("--net:%s=%s", n.OVFName, MorefArg(n.TargetRef)))
	}

	keys := make([]string, 0, len(spec.Properties))
	for k := range spec.Properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if v := spec.Properties[k]; v != "" {
			args = append(args, fmt.Sprintf("--prop:%s=%s", k, v))
		}
	}

	if spec.ExtraArgs != "" {
		args = append(args, strings.Fields(spec.ExtraArgs)...)
	}

	args = append(args, spec.SourcePath, TargetLocator(spec))
	return args
}

// TargetLocator builds the `vi://user:pass@host[:port][/datacenter/host/
// computeResource[/Resources/pool]]` target locator ovftool expects. The
// path is left empty when DatacenterName/ComputeResourceName aren't set,
// which the ovftool docs require when connecting directly to an ESXi host.
func TargetLocator(spec Spec) string {
	// ovftool's vi:// locator is parsed by ovftool itself, not net/url, and
	// its own examples show plain "user:pass@host" with no percent-encoding
	// — so credentials are concatenated raw rather than through
	// url.Userinfo, which would escape characters like '@' that are common
	// in vCenter SSO usernames (e.g. administrator@vsphere.local).
	userinfo := spec.Username + ":" + spec.Password

	host := spec.Hostname
	if spec.Port != 0 {
		host = fmt.Sprintf("%s:%d", spec.Hostname, spec.Port)
	}

	// Always exactly one slash after the host: bare "vi://user:pass@host/"
	// for a direct ESXi connection (confirmed against a live target), or
	// "vi://user:pass@host/DC/host/Cluster[/Resources/Pool]" for vCenter.
	path := ""
	if spec.DatacenterName != "" && spec.ComputeResourceName != "" {
		path = fmt.Sprintf("%s/host/%s", spec.DatacenterName, spec.ComputeResourceName)
		if spec.ResourcePoolName != "" {
			path += "/Resources/" + spec.ResourcePoolName
		}
	}

	return fmt.Sprintf("vi://%s@%s/%s", userinfo, host, path)
}

// RedactedCommandLine renders ovftoolPath+args as a shell-quotable string
// with the target locator's password masked, safe to display in the UI.
func RedactedCommandLine(ovftoolPath string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, quoteArg(ovftoolPath))
	for _, a := range args {
		parts = append(parts, quoteArg(redactLocator(a)))
	}
	return strings.Join(parts, " ")
}

func redactLocator(arg string) string {
	if !strings.HasPrefix(arg, "vi://") {
		return arg
	}
	at := strings.Index(arg, "@")
	if at < 0 {
		return arg
	}
	userinfo := arg[len("vi://"):at]
	if colon := strings.Index(userinfo, ":"); colon >= 0 {
		return "vi://" + userinfo[:colon] + ":***" + arg[at:]
	}
	return arg
}

func quoteArg(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, " \t\"'") {
		return s
	}
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}
