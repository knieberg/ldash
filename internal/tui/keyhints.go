package tui

import "strings"

type keyAction struct {
	Key   string
	Label string
}

func keyHint(key, label string) string {
	return key + " " + label
}

func joinHints(actions []keyAction, max int) string {
	if max <= 0 || max >= len(actions) {
		parts := make([]string, len(actions))
		for i, a := range actions {
			parts[i] = keyHint(a.Key, a.Label)
		}
		return strings.Join(parts, " · ")
	}
	parts := make([]string, max)
	for i := 0; i < max; i++ {
		parts[i] = keyHint(actions[i].Key, actions[i].Label)
	}
	return strings.Join(parts, " · ") + " · … · ? help"
}

func footerCommonBackHelp() []keyAction {
	return []keyAction{{keyBack, "back"}, {keyHelp, "help"}}
}

func footerUsersList(filterOn bool) string {
	actions := []keyAction{
		{keyUp + "/" + keyDown, "move"},
		{keySearch, "search"},
		{keyCreate, "create"},
		{keyEdit, "edit"},
		{keyDelete, "delete"},
		{keyPassword, "password"},
		{keyMail, "mail"},
		{keySamba, "samba"},
		{keyRefresh, "refresh"},
	}
	base := joinHints(actions, 0)
	suffix := joinHints(footerCommonBackHelp(), 0)
	if filterOn {
		return base + " · " + suffix + " (filter on)"
	}
	return base + " · " + suffix
}

func footerGroupsList() string {
	actions := []keyAction{
		{keyUp + "/" + keyDown, "move"},
		{keySearch, "search"},
		{keyCreate, "create"},
		{keyEdit, "edit"},
		{keyDelete, "delete"},
		{keyMembers, "members"},
		{keyRefresh, "refresh"},
	}
	return joinHints(actions, 0) + " · " + joinHints(footerCommonBackHelp(), 0)
}

func footerForm() string {
	return joinHints([]keyAction{
		{"Tab", "next"},
		{keyEnter, "submit"},
		{keyBack, "cancel"},
		{keyHelp, "help"},
	}, 0)
}

func footerDeleteConfirm() string {
	return joinHints([]keyAction{
		{"y", "confirm"},
		{"n", "cancel"},
		{keyHelp, "help"},
	}, 0)
}

func footerLDIFHub() string {
	return joinHints([]keyAction{
		{keyUp + "/" + keyDown, "move"},
		{keyEnter, "open"},
		{keyBack, "back"},
		{keyHelp, "help"},
	}, 0)
}

func footerMenu() string {
	return joinHints([]keyAction{
		{keyUp + "/" + keyDown, "move"},
		{"1-7", "open"},
		{keyEnter, "open"},
		{keyHelp, "help"},
		{keyQuit, "quit"},
	}, 0)
}

func helpLines(actions []keyAction) string {
	var b strings.Builder
	for _, a := range actions {
		b.WriteString(fmtKeyHelp(a.Key, a.Label))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func fmtKeyHelp(key, label string) string {
	return key + "  " + label
}
