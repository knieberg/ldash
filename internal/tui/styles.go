package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Dark admin ANSI-256 tokens (slate + cyan accent, no purple selection).
var (
	colorTitle   = lipgloss.Color("75")
	colorAccent  = lipgloss.Color("81")
	colorSelFG   = lipgloss.Color("229")
	colorSelBG   = lipgloss.Color("24")
	colorOK      = lipgloss.Color("42")
	colorErr     = lipgloss.Color("196")
	colorWarn    = lipgloss.Color("214")
	colorMuted   = lipgloss.Color("247")
	colorBorder  = lipgloss.Color("240")
	colorFooterBG = lipgloss.Color("235")
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorTitle)
	okStyle    = lipgloss.NewStyle().Foreground(colorOK)
	errStyle   = lipgloss.NewStyle().Foreground(colorErr)
	mutedStyle = lipgloss.NewStyle().Foreground(colorMuted)
	boxStyle   = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2)
	selStyle = lipgloss.NewStyle().
			Foreground(colorSelFG).
			Background(colorSelBG)
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	warnStyle   = lipgloss.NewStyle().Foreground(colorWarn)
	crumbStyle  = lipgloss.NewStyle().Foreground(colorMuted)
	crumbActiveStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	disabledStyle    = lipgloss.NewStyle().Foreground(colorMuted)
	panelStyle       = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorBorder).
				Padding(1, 2)
	helpPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(1, 2)
	footerBarStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Background(colorFooterBG).
			Padding(0, 1)
	ruleStyle = lipgloss.NewStyle().Foreground(colorBorder)
)

const (
	formInputMaxWidth = 100
	minAppWidth       = 40
)

func appWidth(termWidth int) int {
	if termWidth <= 0 {
		return 80
	}
	w := termWidth - 4
	if w < minAppWidth {
		return minAppWidth
	}
	return w
}

func formInputWidth(termWidth int) int {
	w := appWidth(termWidth) - 20
	if w < 20 {
		w = 20
	}
	if w > formInputMaxWidth {
		w = formInputMaxWidth
	}
	return w
}

func horizontalRule(width int) string {
	if width < 1 {
		width = 1
	}
	return ruleStyle.Render(strings.Repeat("─", width))
}

type statusKind int

const (
	statusInfo statusKind = iota
	statusOK
	statusError
	statusWarn
	statusLoading
)

func statusLine(kind statusKind, msg string) string {
	if msg == "" {
		return ""
	}
	switch kind {
	case statusOK:
		return okStyle.Render("OK: " + msg)
	case statusError:
		return errStyle.Render("Error: " + msg)
	case statusWarn:
		return warnStyle.Render("Warn: " + msg)
	case statusLoading:
		return mutedStyle.Render("Loading: " + msg)
	default:
		return mutedStyle.Render(msg)
	}
}

func renderFooterBar(keys, status string, width int) string {
	if width < 1 {
		width = 80
	}
	left := keys
	right := status
	gap := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		// Prefer keys; truncate status if needed.
		maxRight := width - lipgloss.Width(left) - 3
		if maxRight < 8 {
			right = truncRunes(right, 8)
		} else {
			right = truncRunes(right, maxRight)
		}
		gap = width - lipgloss.Width(left) - lipgloss.Width(right) - 2
		if gap < 1 {
			gap = 1
		}
	}
	line := left + strings.Repeat(" ", gap) + right
	return footerBarStyle.Width(width).Render(line)
}

func truncRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}

func panel(content string, width int) string {
	if width < minAppWidth {
		width = minAppWidth
	}
	return panelStyle.Width(width).Render(content)
}

func helpBox(content string, width int) string {
	if width < minAppWidth {
		width = minAppWidth
	}
	max := width
	if max > 72 {
		max = 72
	}
	return helpPanelStyle.Width(max).Render(content)
}

func padColumns(cols []string, widths []int) string {
	parts := make([]string, len(cols))
	for i, c := range cols {
		w := 8
		if i < len(widths) {
			w = widths[i]
		}
		parts[i] = fmt.Sprintf("%-*s", w, truncRunes(c, w))
	}
	return strings.Join(parts, " ")
}
