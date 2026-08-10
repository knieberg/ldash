package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	ldifpkg "github.com/knieberg/ldash/internal/ldif"
)

const (
	ldifStepHub = iota
	ldifStepPath
	ldifStepConfirm
	ldifStepSummary
)

func (m model) viewLDIF() string {
	switch m.ldifStep {
	case ldifStepHub:
		opts := []string{"Export LDIF", "Import LDIF"}
		var b strings.Builder
		b.WriteString(headerStyle.Render("LDIF backup and restore"))
		b.WriteString("\n\n")
		for i, o := range opts {
			prefix := "  "
			if i == m.ldifCursor {
				prefix = "▸ "
				b.WriteString(selStyle.Render(prefix + o))
			} else {
				b.WriteString(prefix + o)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render("Export redacts password hashes. Import skips password attributes."))
		return b.String()
	case ldifStepPath:
		var b strings.Builder
		if m.ldifAction == "export" {
			b.WriteString(headerStyle.Render("Export LDIF"))
			b.WriteString("\n\nScope: ")
			scopes := []string{"People", "Groups", "Both"}
			for i, s := range scopes {
				if i == m.ldifScopeIdx {
					b.WriteString(selStyle.Render(s))
				} else {
					b.WriteString(s)
				}
				if i < len(scopes)-1 {
					b.WriteString(" · ")
				}
			}
			b.WriteString("\n\nPath:\n")
		} else {
			b.WriteString(headerStyle.Render("Import LDIF"))
			b.WriteString("\n\nPath:\n")
		}
		b.WriteString(m.ldifPathInput.View())
		return b.String()
	case ldifStepConfirm:
		var b strings.Builder
		b.WriteString(warnStyle.Render("Import LDIF?"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("File: %s\n", m.ldifPath))
		b.WriteString(fmt.Sprintf("Entries: %d\n\n", m.ldifPreview))
		if m.confirm {
			b.WriteString(errStyle.Render("Final confirmation: press y to apply changes."))
		} else {
			b.WriteString("Press y to continue, n to cancel.")
		}
		return b.String()
	case ldifStepSummary:
		var b strings.Builder
		b.WriteString(headerStyle.Render("LDIF result"))
		b.WriteString("\n\n")
		if m.ldifResult != nil {
			fmt.Fprintf(&b, "Applied: %d  Failed: %d  Skipped: %d\n", m.ldifResult.Applied, m.ldifResult.Failed, m.ldifResult.Skipped)
			for i, e := range m.ldifResult.Errors {
				if i >= 5 {
					b.WriteString(mutedStyle.Render(fmt.Sprintf("… and %d more", len(m.ldifResult.Errors)-5)))
					break
				}
				b.WriteString("  " + e + "\n")
			}
		} else if m.ldifExportCount > 0 {
			fmt.Fprintf(&b, "Exported %d entries to %s\n", m.ldifExportCount, m.ldifPath)
		} else {
			b.WriteString(mutedStyle.Render("No entries applied or exported."))
			b.WriteString("\n")
			b.WriteString(mutedStyle.Render("Check skipped/failed counts above or try another file/scope."))
		}
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render("Press Esc to return to LDIF menu."))
		return b.String()
	}
	return ""
}

func defaultExportPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, fmt.Sprintf("ldash-export-%s.ldif", time.Now().Format("20060102")))
}

func (m model) ldifScopeName() string {
	switch m.ldifScopeIdx {
	case 0:
		return "people"
	case 1:
		return "groups"
	default:
		return "both"
	}
}

func (m model) ldifPreviewCmd() tea.Cmd {
	path := m.ldifPath
	return func() tea.Msg {
		n, err := ldifpkg.PreviewImport(path)
		return ldifPreviewMsg{count: n, err: err}
	}
}

func (m model) ldifImportCmd() tea.Cmd {
	path := m.ldifPath
	client := m.client
	return func() tea.Msg {
		res, err := ldifpkg.Import(client, path)
		return ldifDoneMsg{result: res, err: err}
	}
}

func (m model) ldifExportCmd() tea.Cmd {
	path := m.ldifPath
	scope := m.ldifScopeName()
	cfg := m.cfg
	client := m.client
	return func() tea.Msg {
		n, err := ldifpkg.Export(client, cfg, scope, path)
		return ldifDoneMsg{count: n, err: err}
	}
}

func (m model) updateLDIF(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.ldifStep {
	case ldifStepHub:
		switch msg.String() {
		case keyUp, "up":
			if m.ldifCursor > 0 {
				m.ldifCursor--
			}
		case keyDown, "down":
			if m.ldifCursor < 1 {
				m.ldifCursor++
			}
		case keyEnter:
			if m.ldifCursor == 0 {
				m.ldifAction = "export"
			} else {
				m.ldifAction = "import"
			}
			m.ldifStep = ldifStepPath
			m.ldifPath = defaultExportPath()
			m.ldifPathInput.SetValue(m.ldifPath)
			m.ldifPathInput.Width = formInputWidth(m.width)
			m.ldifPathInput.Focus()
			return m, textinput.Blink
		}
	case ldifStepPath:
		return m.updateLDIFPath(msg)
	}
	return m, nil
}

func (m model) updateLDIFPath(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEnter:
		m.ldifPath = strings.TrimSpace(m.ldifPathInput.Value())
		if m.ldifPath == "" {
			m.setStatus(statusError, "path is required")
			return m, nil
		}
		m.ldifPathInput.Blur()
		if m.ldifAction == "import" {
			m.busy = true
			m.setStatus(statusLoading, "Reading LDIF...")
			return m, m.ldifPreviewCmd()
		}
		m.busy = true
		m.setStatus(statusLoading, "Exporting LDIF...")
		return m, m.ensureConnAnd(m.ldifExportCmd())
	case "left", "h":
		if m.ldifAction == "export" && m.ldifScopeIdx > 0 {
			m.ldifScopeIdx--
		}
	case "right", "l":
		if m.ldifAction == "export" && m.ldifScopeIdx < 2 {
			m.ldifScopeIdx++
		}
	}
	var cmd tea.Cmd
	m.ldifPathInput, cmd = m.ldifPathInput.Update(msg)
	return m, cmd
}

func (m model) updateLDIFConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "n":
		m.ldifStep = ldifStepPath
		m.confirm = false
		return m, nil
	case "y":
		if !m.confirm {
			m.confirm = true
			m.setStatus(statusWarn, "Press y again to confirm import")
			return m, nil
		}
		m.busy = true
		m.setStatus(statusLoading, "Importing LDIF...")
		return m, m.ensureConnAnd(m.ldifImportCmd())
	}
	return m, nil
}
