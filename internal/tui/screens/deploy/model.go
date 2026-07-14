// Package deploywizard implements the OVF/OVA deploy wizard: pick a
// source (URL download or local file browser), read its metadata, choose
// target settings and network mappings, configure OVF properties, review
// the generated ovftool command, and watch it run live.
package deploywizard

import (
	"context"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/mousavi-azure/ovftop/internal/config"
	"github.com/mousavi-azure/ovftop/internal/deploy"
	"github.com/mousavi-azure/ovftop/internal/deploy/ovf"
	"github.com/mousavi-azure/ovftop/internal/tui/components"
	"github.com/mousavi-azure/ovftop/internal/tui/theme"
	"github.com/mousavi-azure/ovftop/internal/vsphere"
)

type step int

const (
	stepSource step = iota
	stepMetadata
	stepSettings
	stepNetwork
	stepAdvanced
	stepSummary
	stepProgress
)

// ClosedMsg is emitted when the user backs out of the wizard entirely
// (Esc from the first step), asking the parent app to return to the
// dashboard.
type ClosedMsg struct{}

type sourceMethod int

const (
	methodURL sourceMethod = iota
	methodBrowse
)

// refOption is one entry in a datastore/network/resource-pool picklist.
type refOption struct {
	Name string
	Ref  types.ManagedObjectReference
}

// computeOption is one entry in the target compute-resource (cluster or
// standalone host) picklist, along with the datacenter it belongs to.
type computeOption struct {
	DatacenterName string
	Name           string
	Ref            types.ManagedObjectReference
}

// Model is the deploy wizard's root state machine.
type Model struct {
	styles theme.Styles
	cfg    *config.Manager

	profile   config.ConnectionProfile
	password  string
	inventory *vsphere.Inventory
	isESXi    bool

	step step
	err  error

	// Step 1: source
	sourceMethod  sourceMethod
	urlInput      textinput.Model
	browser       *components.FileBrowser
	downloading   bool
	downloadState *downloadState
	downloadInfo  deploy.DownloadProgress
	sourcePath    string

	// Step 2: metadata
	probing    bool
	descriptor *ovf.Descriptor

	// Step 3: settings
	vmNameInput      textinput.Model
	computeOptions   []computeOption
	computeCursor    int
	datastoreOptions []refOption
	datastoreCursor  int
	diskModeThick    bool
	powerOn          bool
	importAsTemplate bool
	settingsFocus    int

	// Step 4: network mapping
	networkTargets []refOption
	networkMap     []int // parallel to descriptor.Networks; index into networkTargets, -1 = unmapped
	networkCursor  int

	// Step 5: advanced (OVF properties + extra args)
	propertyInputs []textinput.Model
	extraArgsInput textinput.Model
	advancedFocus  int

	// Step 6: summary
	spec    deploy.Spec
	args    []string
	command string

	// Step 7: progress
	running   bool
	log       viewport.Model
	logLines  []string
	events    chan deploy.Event
	percent   int
	lastLine  string
	startTime time.Time
	elapsed   time.Duration
	cancel    context.CancelFunc
	deployErr error
	done      bool
	succeeded bool

	width, height int
}

// New creates a deploy wizard scoped to the currently connected endpoint.
// password is the plaintext credential for that connection, needed to
// build the ovftool target locator at summary/deploy time.
func New(styles theme.Styles, cfg *config.Manager, profile config.ConnectionProfile, password string, inv *vsphere.Inventory) *Model {
	m := &Model{
		styles:    styles,
		cfg:       cfg,
		profile:   profile,
		password:  password,
		inventory: inv,
		isESXi:    inv.Summary.ConnectionType == "ESXi",
	}

	m.urlInput = textinput.New()
	m.urlInput.Placeholder = "https://example.com/appliance.ova"
	m.urlInput.Focus()

	home, _ := os.UserHomeDir()
	m.browser = components.NewFileBrowser(styles, home)

	m.vmNameInput = textinput.New()
	m.vmNameInput.Placeholder = "my-new-vm"
	m.vmNameInput.CharLimit = 80

	m.extraArgsInput = textinput.New()
	m.extraArgsInput.Placeholder = "--lax --diskSize:vm1=1024"

	m.computeOptions, m.datastoreOptions, m.networkTargets = collectOptions(inv)

	m.log = viewport.New(80, 20)

	return m
}

// collectOptions flattens the discovered inventory into flat picklists
// for the settings/network steps.
func collectOptions(inv *vsphere.Inventory) (compute []computeOption, datastores, networks []refOption) {
	for _, dc := range inv.Root {
		for _, group := range dc.Children {
			if group.Kind != vsphere.KindGroup {
				continue
			}
			switch group.Name {
			case "Hosts & Clusters":
				for _, n := range group.Children {
					if n.Kind == vsphere.KindCluster || n.Kind == vsphere.KindHost {
						compute = append(compute, computeOption{DatacenterName: dc.Name, Name: n.Name, Ref: n.Ref})
					}
				}
			case "Datastores":
				for _, n := range group.Children {
					datastores = append(datastores, refOption{Name: n.Name, Ref: n.Ref})
				}
			case "Networks":
				for _, n := range group.Children {
					networks = append(networks, refOption{Name: n.Name, Ref: n.Ref})
				}
			}
		}
	}
	return compute, datastores, networks
}

// Init satisfies the tea.Model-shaped contract used by the parent app.
func (m *Model) Init() tea.Cmd { return textinput.Blink }

// SetSize updates the viewport dimensions.
func (m *Model) SetSize(w, h int) {
	m.width, m.height = w, h
	m.log.Width = w - 4
	m.log.Height = h - 12
	if m.browser != nil {
		m.browser.SetSize(w-4, h-14)
	}
}

// Update dispatches to the current step's handler.
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" && m.step == stepSource && !m.downloading {
			return m, func() tea.Msg { return ClosedMsg{} }
		}
	}

	switch m.step {
	case stepSource:
		return m.updateSource(msg)
	case stepMetadata:
		return m.updateMetadata(msg)
	case stepSettings:
		return m.updateSettings(msg)
	case stepNetwork:
		return m.updateNetwork(msg)
	case stepAdvanced:
		return m.updateAdvanced(msg)
	case stepSummary:
		return m.updateSummary(msg)
	case stepProgress:
		return m.updateProgress(msg)
	}
	return m, nil
}

// View renders the current step.
func (m *Model) View() string {
	switch m.step {
	case stepSource:
		return m.viewSource()
	case stepMetadata:
		return m.viewMetadata()
	case stepSettings:
		return m.viewSettings()
	case stepNetwork:
		return m.viewNetwork()
	case stepAdvanced:
		return m.viewAdvanced()
	case stepSummary:
		return m.viewSummary()
	case stepProgress:
		return m.viewProgress()
	}
	return ""
}

func (m *Model) titleBar() string {
	return components.RenderTitleBar(m.styles, m.width, "🚀", "Deploy OVF/OVA", "")
}

var stepNames = []string{"1 Source", "2 Metadata", "3 Settings", "4 Network", "5 Advanced", "6 Summary", "7 Deploy"}

func (m *Model) stepper() string {
	var out string
	for i, name := range stepNames {
		if step(i) == m.step {
			out += m.styles.KeyHint.Render(name) + " "
		} else if step(i) < m.step {
			out += m.styles.Success.Render("✓"+name) + " "
		} else {
			out += m.styles.KeyHintOff.Render(name) + " "
		}
	}
	return out
}
