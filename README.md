<div align="center">

```
  ______      ________ _______ ____  _____
 / __ \ \    / /  ____|__   __/ __ \|  __ \
| |  | \ \  / /| |__     | | | |  | | |__) |
| |  | |\ \/ / |  __|    | | | |  | |  ___/
| |__| | \  /  | |       | | | |__| | |
 \____/   \/   |_|       |_|  \____/|_|
```

**A terminal UI for deploying and managing OVF/OVA virtual machines on VMware ESXi and vCenter.**

Midnight-Commander-style dashboard · guided deploy wizard · one-key clone & export · zero config server required

[![CI](https://github.com/mousavi-azure/ovftop/actions/workflows/ci.yml/badge.svg)](https://github.com/mousavi-azure/ovftop/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/mousavi-azure/ovftop)](https://github.com/mousavi-azure/ovftop/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/mousavi-azure/ovftop.svg)](https://pkg.go.dev/github.com/mousavi-azure/ovftop)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/mousavi-azure/ovftop)](https://goreportcard.com/report/github.com/mousavi-azure/ovftop)

</div>

---

## Why OVFTOP

Deploying an OVF/OVA to ESXi or vCenter usually means one of two things: clicking through the vSphere web client's import wizard for the hundredth time, or hand-assembling an `ovftool` command line and hoping you remembered every `--net:` and `--prop:` flag correctly.

OVFTOP is a fast, keyboard-driven terminal UI that sits on top of VMware's own `ovftool`. It gives you a live, browsable inventory of your host or vCenter, a guided step-by-step deploy wizard that builds the `ovftool` invocation for you (and lets you review it before anything runs), and one-key clone/export for VMs and templates — all without leaving the terminal, and without standing up any kind of server or agent.

If you've used `htop`, `btop`, `k9s`, or `lazygit`, you already know how to drive this.

## Features

- **Live infrastructure tree** — datacenters, clusters, hosts, VMs, templates, datastores, networks, and resource pools, in a collapsible Midnight-Commander-style tree with live CPU/memory/datastore usage gauges.
- **Guided deploy wizard** — a 7-step flow (Source → Metadata → Settings → Network → Advanced → Summary → Deploy) that reads the OVF's own descriptor to prompt for exactly the properties and network mappings it declares, then shows you the full `ovftool` command before it runs.
- **Deploy from a local file or a URL** — point it at a local `.ovf`/`.ova`, or give it a download URL and it fetches the file for you first.
- **One-key clone** (`F5`) — clone a selected VM or template to a new VM without opening a wizard.
- **One-key export** (`F6`) — export a selected VM/template back out to a `.ova` file, with live progress.
- **Live deploy/export progress** — streamed `ovftool` output with a progress bar, not just a spinner.
- **Both ESXi and vCenter** — works against a standalone ESXi host or a full vCenter inventory.
- **Saved connection profiles** — with credentials encrypted at rest (AES-GCM, scrypt-derived key); see [Credential storage](#credential-storage).
- **Auto-refresh** — the dashboard re-polls inventory on a configurable interval (off / 1 / 2 / 5 / 10 minutes) so long sessions stay in sync.
- **Four built-in themes** — Dark, Light, Dracula, Nord — cycle with one keypress from Settings.
- **Built-in help & activity log** — every keybinding is one `F1` away; every connect/deploy/error is logged and viewable in-app (`F8`).

## Screens at a glance

| Screen | Key | What it does |
|---|---|---|
| Dashboard | — | Infrastructure tree + details panel + live usage gauges |
| Deploy wizard | `F4` | Guided OVF/OVA import, source → summary → live progress |
| Clone | `F5` | Clone the selected VM/template to a new name |
| Export | `F6` | Export the selected VM/template to a `.ova` |
| Settings | `F7` | Theme, auto-refresh interval, config/log paths |
| Logs | `F8` | Scrollable view of the app's own activity log |
| Menu | `F9` | Discoverable list of the actions above, plus About |
| Help | `F1` | Full keybinding reference |

## Requirements

- **VMware's official `ovftool`**, installed separately and on your `PATH`. OVFTOP drives `ovftool`; it doesn't replace it, bundle it, or reimplement OVF parsing. Download it from [VMware/Broadcom's support portal](https://developer.broadcom.com/tools/open-virtualization-format-ovf-tool/latest) (free registration required) and install it for your OS.

  > **Why isn't this in the repo?** `ovftool` is proprietary software under VMware's own EULA — this project can't legally redistribute it. `install.sh` in this repo is a convenience script that installs *your own* downloaded copy on Linux; it doesn't fetch it for you.

- Network access to the ESXi host or vCenter Server you want to manage, plus an account with permission to import/export/clone VMs there.
- Linux, macOS, or Windows. A terminal with reasonable Unicode/emoji support looks best (the tree and status bar use icons as status indicators), but everything is still fully legible without it.

## Installation

### Download a prebuilt binary (recommended)

Grab the archive for your OS/arch from the [latest release](https://github.com/mousavi-azure/ovftop/releases/latest), extract it, and put the `ovftop` binary on your `PATH`:

```sh
# Linux (amd64) example — swap in your OS/arch
curl -L https://github.com/mousavi-azure/ovftop/releases/latest/download/ovftop_<version>_linux_amd64.tar.gz | tar xz
sudo install -m 0755 ovftop /usr/local/bin/ovftop
```

### Build from source

Requires Go 1.24+.

```sh
go install github.com/mousavi-azure/ovftop/cmd/ovftop@latest
```

or clone and build locally:

```sh
git clone https://github.com/mousavi-azure/ovftop.git
cd ovftop
go build -o ovftop ./cmd/ovftop
```

### Install `ovftool`

```sh
# Linux, using this repo's helper against your own downloaded copy
sudo ./install.sh /path/to/VMware-ovftool-<version>-lin.x86_64.zip
```

On macOS/Windows, run VMware's own installer for `ovftool`.

## Quick start

```sh
ovftop
```

1. On first run you'll land on the **connect** screen — press `n` to add a connection profile (ESXi host or vCenter Server, hostname, credentials).
2. On connect, OVFTOP discovers the full inventory and drops you on the **dashboard**.
3. Navigate the tree with `↑`/`↓`, expand/collapse with `Enter`/`Space`.
4. Press `F4` to deploy an OVF/OVA, `F5` to clone a selected VM/template, `F6` to export one, `F1` any time for the full keybinding reference.

## Configuration & credential storage

OVFTOP stores its config, log, and encrypted credential vault in your OS's standard per-user config directory (e.g. `~/.config/ovftop` on Linux, `~/Library/Application Support/ovftop` on macOS):

| File | Contents |
|---|---|
| `config.yaml` | Saved connection profiles (no secrets) |
| `vault.enc` | AES-GCM–encrypted saved passwords |
| `vault.key` / `vault.salt` | Local key material for the vault (0600 permissions) |
| `ovftop.log` | Recent activity log, viewable in-app via `F8` |

By default the vault's encryption key is a random 32-byte key generated once and stored locally with `0600` permissions — this protects against casual copying of the config directory, not against another process running as the same OS user. If you'd rather derive the key from a passphrase you supply on every launch instead, set:

```sh
export OVFTOP_MASTER_PASSWORD="your passphrase"
```

## Building a release locally

Releases are built with [GoReleaser](https://goreleaser.com) from `.goreleaser.yaml`. To build all platform archives locally without publishing:

```sh
goreleaser release --snapshot --clean
```

## Contributing

Issues and PRs are welcome. Before opening a PR:

```sh
go build ./...
go vet ./...
go test ./...
gofmt -l .   # should print nothing
```

## Acknowledgements

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Lip Gloss](https://github.com/charmbracelet/lipgloss), and [Bubbles](https://github.com/charmbracelet/bubbles) from Charm, [govmomi](https://github.com/vmware/govmomi) for the vSphere API client, and [Cobra](https://github.com/spf13/cobra) for the CLI shell. Drives — but does not include — VMware's official `ovftool`.

## License

[MIT](LICENSE) © 2026 Mostafa Mousavi — [mousavi.dev](https://mousavi.dev)

"OVF Tool" and "vSphere" are trademarks of VMware/Broadcom. This is an independent, unaffiliated project.
