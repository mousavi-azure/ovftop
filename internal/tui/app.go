// Package tui wires the individual screens (connection wizard, main
// dashboard, deploy wizard, logs, menu) into a single Bubble Tea program,
// owning the active vSphere session and dispatching background work
// (connect, discover, refresh) as commands.
package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/mousavi-azure/ovftop/internal/config"
	"github.com/mousavi-azure/ovftop/internal/logging"
	"github.com/mousavi-azure/ovftop/internal/tui/screens/connect"
	deploywizard "github.com/mousavi-azure/ovftop/internal/tui/screens/deploy"
	"github.com/mousavi-azure/ovftop/internal/tui/screens/export"
	"github.com/mousavi-azure/ovftop/internal/tui/screens/help"
	"github.com/mousavi-azure/ovftop/internal/tui/screens/logs"
	dashboard "github.com/mousavi-azure/ovftop/internal/tui/screens/main"
	"github.com/mousavi-azure/ovftop/internal/tui/screens/menu"
	"github.com/mousavi-azure/ovftop/internal/tui/screens/settings"
	"github.com/mousavi-azure/ovftop/internal/tui/theme"
	"github.com/mousavi-azure/ovftop/internal/vsphere"
)

const discoveryTimeout = 30 * time.Second

type screen int

const (
	screenConnect screen = iota
	screenDashboard
	screenDeploy
	screenLogs
	screenMenu
	screenHelp
	screenSettings
	screenExport
)

type connectResultMsg struct {
	client    *vsphere.Client
	inventory *vsphere.Inventory
	profile   config.ConnectionProfile
	password  string
	err       error
}

type refreshResultMsg struct {
	inventory *vsphere.Inventory
	err       error
}

type cloneResultMsg struct {
	name string
	err  error
}

// App is the root Bubble Tea model. It owns the active vSphere client
// (if any) and routes messages to whichever screen is currently active.
type App struct {
	cfg    *config.Manager
	logger *logging.Logger
	styles theme.Styles

	screen         screen
	prevScreen     screen
	connectScreen  *connect.Model
	dashboard      *dashboard.Model
	deployScreen   *deploywizard.Model
	logsScreen     *logs.Model
	menuScreen     *menu.Model
	helpScreen     *help.Model
	settingsScreen *settings.Model
	exportScreen   *export.Model

	client          *vsphere.Client
	activeProfile   config.ConnectionProfile
	password        string
	connectionLabel string
	inventory       *vsphere.Inventory

	width, height int
}

// NewApp constructs the root model, ready to run via tea.NewProgram.
func NewApp(cfg *config.Manager, logger *logging.Logger) *App {
	styles := theme.New(theme.Get(cfg.Preferences().Theme))
	db := dashboard.New(styles)
	db.SetRefreshInterval(time.Duration(cfg.Preferences().RefreshIntervalSeconds) * time.Second)
	return &App{
		cfg:           cfg,
		logger:        logger,
		styles:        styles,
		screen:        screenConnect,
		connectScreen: connect.New(styles, cfg),
		dashboard:     db,
	}
}

// Init starts the connection wizard.
func (a *App) Init() tea.Cmd { return a.connectScreen.Init() }

// Update dispatches messages: window resizes and cross-screen navigation
// are handled here; everything else is delegated to the active screen.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.connectScreen.SetSize(msg.Width, msg.Height)
		a.dashboard.SetSize(msg.Width, msg.Height)
		if a.deployScreen != nil {
			a.deployScreen.SetSize(msg.Width, msg.Height)
		}
		if a.logsScreen != nil {
			a.logsScreen.SetSize(msg.Width, msg.Height)
		}
		if a.menuScreen != nil {
			a.menuScreen.SetSize(msg.Width, msg.Height)
		}
		if a.helpScreen != nil {
			a.helpScreen.SetSize(msg.Width, msg.Height)
		}
		if a.settingsScreen != nil {
			a.settingsScreen.SetSize(msg.Width, msg.Height)
		}
		if a.exportScreen != nil {
			a.exportScreen.SetSize(msg.Width, msg.Height)
		}
		return a, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, a.quit()
		}

	case connect.ConnectRequestedMsg:
		a.logger.Info("connect", "connecting to %s@%s", msg.Profile.Username, msg.Profile.Hostname)
		return a, connectCmd(msg.Profile, msg.Password)

	case connectResultMsg:
		if msg.err != nil {
			a.logger.Error("connect", "failed to connect to %s: %v", msg.profile.Hostname, msg.err)
			a.connectScreen.SetError(msg.err)
			return a, nil
		}
		a.logger.Info("connect", "connected to %s (%s)", msg.profile.Hostname, msg.inventory.Summary.ConnectionType)
		a.client = msg.client
		a.activeProfile = msg.profile
		a.password = msg.password
		a.connectionLabel = msg.inventory.Summary.ConnectionType
		a.inventory = msg.inventory
		a.dashboard.SetInventory(msg.inventory, a.connectionLabel, msg.profile.Hostname)
		a.screen = screenDashboard
		return a, a.dashboard.Tick()

	case dashboard.RefreshRequestedMsg:
		return a, refreshCmd(a.client)

	case refreshResultMsg:
		if msg.err != nil {
			a.logger.Warn("refresh", "discovery failed: %v", msg.err)
			a.dashboard.SetError(msg.err)
			return a, nil
		}
		a.inventory = msg.inventory
		a.dashboard.SetInventory(msg.inventory, a.connectionLabel, a.activeProfile.Hostname)
		return a, nil

	case dashboard.DisconnectRequestedMsg:
		a.logger.Info("connect", "disconnected from %s", a.activeProfile.Hostname)
		a.disconnect()
		a.screen = screenConnect
		return a, nil

	case dashboard.QuitRequestedMsg:
		return a, a.quit()

	case dashboard.DeployRequestedMsg:
		a.deployScreen = deploywizard.New(a.styles, a.cfg, a.activeProfile, a.password, a.inventory)
		a.deployScreen.SetSize(a.width, a.height)
		a.screen = screenDeploy
		return a, a.deployScreen.Init()

	case deploywizard.ClosedMsg:
		a.deployScreen = nil
		a.screen = screenDashboard
		return a, nil

	case dashboard.LogsRequestedMsg:
		a.openLogs()
		return a, nil

	case logs.ClosedMsg:
		a.screen = screenDashboard
		return a, nil

	case dashboard.HelpRequestedMsg:
		a.helpScreen = help.New(a.styles)
		a.helpScreen.SetSize(a.width, a.height)
		a.prevScreen = a.screen
		a.screen = screenHelp
		return a, nil

	case help.ClosedMsg:
		a.screen = a.prevScreen
		return a, nil

	case dashboard.SettingsRequestedMsg:
		a.openSettings()
		return a, nil

	case settings.ClosedMsg:
		a.screen = screenDashboard
		return a, nil

	case settings.CycleThemeMsg:
		a.applyTheme(theme.Next(a.cfg.Preferences().Theme))
		return a, nil

	case settings.SetRefreshIntervalMsg:
		prefs := a.cfg.Preferences()
		prefs.RefreshIntervalSeconds = msg.Seconds
		_ = a.cfg.SetPreferences(prefs)
		a.dashboard.SetRefreshInterval(time.Duration(msg.Seconds) * time.Second)
		return a, a.dashboard.Tick()

	case dashboard.CloneRequestedMsg:
		a.logger.Info("clone", "cloning to new VM %q", msg.NewName)
		return a, cloneCmd(a.client, msg.Ref, msg.NewName)

	case cloneResultMsg:
		a.dashboard.SetCloning(false)
		if msg.err != nil {
			a.logger.Error("clone", "clone to %q failed: %v", msg.name, msg.err)
			a.dashboard.SetError(msg.err)
			return a, nil
		}
		a.logger.Info("clone", "clone to %q completed", msg.name)
		a.dashboard.SetRefreshing(true)
		return a, refreshCmd(a.client)

	case dashboard.ExportRequestedMsg:
		a.exportScreen = export.New(a.styles, export.Params{
			Hostname:   a.activeProfile.Hostname,
			Port:       a.activeProfile.Port,
			Username:   a.activeProfile.Username,
			Password:   a.password,
			Datacenter: msg.Datacenter,
			VMName:     msg.Name,
		})
		a.exportScreen.SetSize(a.width, a.height)
		a.prevScreen = a.screen
		a.screen = screenExport
		return a, nil

	case export.ClosedMsg:
		a.exportScreen = nil
		a.screen = screenDashboard
		return a, nil

	case dashboard.MenuRequestedMsg:
		a.menuScreen = menu.New(a.styles, a.cfg.Preferences().Theme)
		a.menuScreen.SetSize(a.width, a.height)
		a.prevScreen = a.screen
		a.screen = screenMenu
		return a, nil

	case menu.ClosedMsg:
		a.screen = a.prevScreen
		return a, nil

	case menu.ViewLogsMsg:
		a.openLogs()
		return a, nil

	case menu.CycleThemeMsg:
		a.applyTheme(theme.Next(a.cfg.Preferences().Theme))
		return a, nil

	case menu.RefreshNowMsg:
		a.screen = screenDashboard
		a.dashboard.SetRefreshing(true)
		return a, refreshCmd(a.client)

	case menu.DisconnectMsg:
		a.logger.Info("connect", "disconnected from %s", a.activeProfile.Hostname)
		a.disconnect()
		a.screen = screenConnect
		return a, nil

	case menu.QuitMsg:
		return a, a.quit()
	}

	var cmd tea.Cmd
	switch a.screen {
	case screenDashboard:
		a.dashboard, cmd = a.dashboard.Update(msg)
	case screenDeploy:
		a.deployScreen, cmd = a.deployScreen.Update(msg)
	case screenLogs:
		a.logsScreen, cmd = a.logsScreen.Update(msg)
	case screenMenu:
		a.menuScreen, cmd = a.menuScreen.Update(msg)
	case screenHelp:
		a.helpScreen, cmd = a.helpScreen.Update(msg)
	case screenSettings:
		a.settingsScreen, cmd = a.settingsScreen.Update(msg)
	case screenExport:
		a.exportScreen, cmd = a.exportScreen.Update(msg)
	default:
		a.connectScreen, cmd = a.connectScreen.Update(msg)
	}
	return a, cmd
}

func (a *App) openLogs() {
	a.logsScreen = logs.New(a.styles, a.logger)
	a.logsScreen.SetSize(a.width, a.height)
	a.prevScreen = screenDashboard
	a.screen = screenLogs
}

func (a *App) openSettings() {
	dir, _ := config.Dir()
	logPath, _ := config.LogPath()
	a.settingsScreen = settings.New(a.styles, a.cfg.Preferences().Theme, a.cfg.Preferences().RefreshIntervalSeconds, dir, logPath)
	a.settingsScreen.SetSize(a.width, a.height)
	a.prevScreen = a.screen
	a.screen = screenSettings
}

// applyTheme switches the active color palette, persists the preference,
// and rebuilds every screen that doesn't hold transient in-progress state
// (the deploy wizard and export screen are intentionally left alone —
// swapping styles mid-deploy/mid-export isn't worth the complexity of
// migrating their in-flight state).
func (a *App) applyTheme(name string) {
	a.styles = theme.New(theme.Get(name))
	prefs := a.cfg.Preferences()
	prefs.Theme = name
	_ = a.cfg.SetPreferences(prefs)

	a.connectScreen = connect.New(a.styles, a.cfg)
	a.connectScreen.SetSize(a.width, a.height)

	refreshInterval := a.dashboard.RefreshInterval()
	a.dashboard = dashboard.New(a.styles)
	a.dashboard.SetRefreshInterval(refreshInterval)
	if a.inventory != nil {
		a.dashboard.SetInventory(a.inventory, a.connectionLabel, a.activeProfile.Hostname)
	}
	a.dashboard.SetSize(a.width, a.height)

	if a.logsScreen != nil {
		a.logsScreen = logs.New(a.styles, a.logger)
		a.logsScreen.SetSize(a.width, a.height)
	}
	if a.menuScreen != nil {
		a.menuScreen = menu.New(a.styles, name)
		a.menuScreen.SetSize(a.width, a.height)
	}
	if a.helpScreen != nil {
		a.helpScreen = help.New(a.styles)
		a.helpScreen.SetSize(a.width, a.height)
	}
	if a.settingsScreen != nil {
		dir, _ := config.Dir()
		logPath, _ := config.LogPath()
		a.settingsScreen = settings.New(a.styles, name, a.cfg.Preferences().RefreshIntervalSeconds, dir, logPath)
		a.settingsScreen.SetSize(a.width, a.height)
	}
}

// View renders whichever screen is currently active.
func (a *App) View() string {
	switch a.screen {
	case screenDashboard:
		return a.dashboard.View()
	case screenDeploy:
		return a.deployScreen.View()
	case screenLogs:
		return a.logsScreen.View()
	case screenMenu:
		return a.menuScreen.View()
	case screenHelp:
		return a.helpScreen.View()
	case screenSettings:
		return a.settingsScreen.View()
	case screenExport:
		return a.exportScreen.View()
	default:
		return a.connectScreen.View()
	}
}

func (a *App) disconnect() {
	if a.deployScreen != nil {
		a.deployScreen.CancelActiveDeploy()
	}
	if a.exportScreen != nil {
		a.exportScreen.CancelActiveExport()
	}
	if a.client == nil {
		return
	}
	client := a.client
	a.client = nil
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Close(ctx)
	}()
}

func (a *App) quit() tea.Cmd {
	a.disconnect()
	return tea.Quit
}

func connectCmd(profile config.ConnectionProfile, password string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), discoveryTimeout)
		defer cancel()

		client, err := vsphere.Connect(ctx, vsphere.ConnectParams{
			Hostname:  profile.Hostname,
			Port:      profile.Port,
			Username:  profile.Username,
			Password:  password,
			IgnoreSSL: profile.IgnoreSSL,
		})
		if err != nil {
			return connectResultMsg{err: err, profile: profile}
		}

		inv, err := client.Discover(ctx)
		if err != nil {
			closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = client.Close(closeCtx)
			return connectResultMsg{err: fmt.Errorf("connected, but discovery failed: %w", err), profile: profile}
		}

		return connectResultMsg{client: client, inventory: inv, profile: profile, password: password}
	}
}

func refreshCmd(client *vsphere.Client) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return refreshResultMsg{err: fmt.Errorf("not connected")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), discoveryTimeout)
		defer cancel()
		inv, err := client.Discover(ctx)
		return refreshResultMsg{inventory: inv, err: err}
	}
}

const cloneTimeout = 10 * time.Minute

func cloneCmd(client *vsphere.Client, ref types.ManagedObjectReference, newName string) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return cloneResultMsg{name: newName, err: fmt.Errorf("not connected")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), cloneTimeout)
		defer cancel()
		err := client.CloneVM(ctx, ref, newName)
		return cloneResultMsg{name: newName, err: err}
	}
}
