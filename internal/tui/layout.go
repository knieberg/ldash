package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderShell builds a full-bleed header, height-filled body, and sticky footer.
func renderShell(header, body, footer string, width, height int) string {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		return strings.TrimRight(header+"\n"+body+"\n"+footer, "\n") + "\n"
	}

	headerH := lipgloss.Height(header)
	footerH := lipgloss.Height(footer)
	bodyH := height - headerH - footerH
	if bodyH < 1 {
		bodyH = 1
	}

	bodyBlock := lipgloss.NewStyle().
		Width(width).
		Height(bodyH).
		Align(lipgloss.Left, lipgloss.Top).
		Render(body)

	return lipgloss.JoinVertical(lipgloss.Left, header, bodyBlock, footer)
}

func renderHeader(title, breadcrumb string, width int) string {
	if width < 1 {
		width = 80
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(breadcrumb)
	b.WriteString("\n")
	b.WriteString(horizontalRule(width))
	return b.String()
}
