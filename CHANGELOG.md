# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.1] — 2026-07-16

### Fixed

- Export (`F6`) failing against a standalone ESXi host with `Locator does not refer to an object`: the export locator was including ESXi's implicit `ha-datacenter` pseudo-datacenter in the `vi://` path, which `ovftool` rejects for direct ESXi connections. It now omits the datacenter segment for standalone ESXi, matching the Deploy wizard's existing locator behavior.
- Bottom status bar silently truncating mid-button (e.g. showing `F6` with its `Export` label cut off) on terminals narrower than the full labeled row. It now falls back to a compact key-only row when the labeled row won't fit, instead of chopping a button in half.

## [1.0.0] — 2026-07-14

Initial public release.

### Added

- Terminal dashboard for a connected ESXi host or vCenter Server: collapsible infrastructure tree (datacenters, clusters, hosts, VMs, templates, datastores, networks, resource pools), a details panel for the selected node, and live CPU/memory/datastore usage gauges.
- Guided 7-step deploy wizard (Source → Metadata → Settings → Network → Advanced → Summary → Deploy) that reads an OVF's own descriptor to prompt for its declared properties and network mappings, previews the resulting `ovftool` command line, and streams live deploy progress.
- Deploy from a local `.ovf`/`.ova` file or a download URL.
- One-key clone (`F5`) of a selected VM or template to a new VM.
- One-key export (`F6`) of a selected VM/template to a `.ova` file, with live progress.
- Saved connection profiles for both standalone ESXi hosts and vCenter Server, with passwords encrypted at rest (AES-GCM, scrypt-derived key by default, or passphrase-derived via `OVFTOP_MASTER_PASSWORD`).
- Configurable dashboard auto-refresh (off / 1 / 2 / 5 / 10 minutes).
- Four built-in themes: Dark, Light, Dracula, Nord.
- In-app help screen (`F1`) with a full keybinding reference, and an activity log viewer (`F8`).
- Cross-platform prebuilt binaries (Linux/macOS/Windows, amd64/arm64) published via GitHub Releases.

[1.0.1]: https://github.com/mousavi-azure/ovftop/releases/tag/v1.0.1
[1.0.0]: https://github.com/mousavi-azure/ovftop/releases/tag/v1.0.0
