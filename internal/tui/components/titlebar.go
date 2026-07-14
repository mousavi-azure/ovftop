package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/mousavi-azure/ovftop/internal/tui/theme"
	"github.com/mousavi-azure/ovftop/internal/version"
)

// RenderTitleBar draws the top title bar shared by every screen: a bold
// "OVFTOP vX.Y.Z" wordmark (optionally preceded by a screen icon) plus an
// optional section label for the current screen, rendered at lower weight
// so the product name stays the clear visual anchor across the app. right,
// if non-empty, is right-aligned on the same bar (e.g. author credit on
// the dashboard) and silently dropped if the bar is too narrow to fit it
// without truncating mid-word.
func RenderTitleBar(styles theme.Styles, width int, icon, section, right string) string {
	brand := "OVFTOP v" + version.Version
	if icon != "" {
		brand = icon + " " + brand
	}
	left := styles.TitleBarBrand.Render(brand)
	if section != "" {
		left += styles.TitleBarSection.Render("  ·  " + section)
	}

	if right == "" {
		return styles.TitleBar.Width(width).Render(left)
	}

	const barPadding = 2 // TitleBar's Padding(0, 1) on each side
	gap := width - barPadding - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 2 {
		return styles.TitleBar.Width(width).Render(left)
	}
	fill := styles.TitleBarSection.Render(strings.Repeat(" ", gap))
	return styles.TitleBar.Width(width).Render(left + fill + right)
}

// RenderTitleBarCredit renders the "Created by ... · link" fragment used as
// the right-hand side of the dashboard's title bar, styled to match the
// splash screen's credit line (see RenderBanner).
func RenderTitleBarCredit(styles theme.Styles) string {
	link := styles.TitleBarSection.Underline(true).Render("mousavi.dev")
	return styles.TitleBarSection.Render("Created by Mostafa Mousavi  ·  🔗 ") + link
}
