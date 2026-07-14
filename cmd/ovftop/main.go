// Command ovftop is a terminal UI for deploying and managing OVF/OVA
// virtual machines on VMware ESXi and vCenter via ovftool.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/mousavi-azure/ovftop/internal/config"
	"github.com/mousavi-azure/ovftop/internal/logging"
	"github.com/mousavi-azure/ovftop/internal/tui"
	"github.com/mousavi-azure/ovftop/internal/version"
)

func main() {
	root := &cobra.Command{
		Use:           "ovftop",
		Short:         "OVFTOP — terminal deployment manager for VMware ESXi/vCenter OVF/OVA templates",
		Version:       "v" + version.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading configuration: %w", err)
			}

			logPath, err := config.LogPath()
			if err != nil {
				return fmt.Errorf("resolving log path: %w", err)
			}
			logger, err := logging.Open(logPath)
			if err != nil {
				return fmt.Errorf("opening log file: %w", err)
			}
			defer logger.Close()

			app := tui.NewApp(cfg, logger)
			program := tea.NewProgram(app, tea.WithAltScreen())
			_, err = program.Run()
			return err
		},
	}

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "ovftop:", err)
		os.Exit(1)
	}
}
