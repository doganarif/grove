package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/doganarif/grove/internal/tmux"
)

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  grove: %v\n\n  Press q to quit.\n", m.err)
	}
	if m.loading && len(m.items) == 0 {
		return "\n  Loading worktrees...\n"
	}

	switch m.mode {
	case modeHelp:
		return m.viewHelp()
	case modeAdd:
		return m.viewWithModal(m.viewAddModal())
	case modeDelete:
		return m.viewWithModal(m.viewDeleteModal())
	case modePrune:
		return m.viewWithModal(m.viewPruneModal())
	case modeColor:
		return m.viewWithModal(m.viewColorModal())
	case modeNote:
		return m.viewWithModal(m.viewNoteModal())
	case modeTmux:
		return m.viewWithModal(m.viewTmuxModal())
	}

	return m.viewMain()
}

func (m Model) viewMain() string {
	var b strings.Builder

	// Header
	repoLabel := m.repoName
	if m.isBare {
		repoLabel += " (.bare)"
	}

	agentCount := 0
	for _, it := range m.items {
		if it.Agent != nil && it.Agent.Active {
			agentCount++
		}
	}

	header := styleHeader.Render(" grove") +
		styleMuted.Render("  "+repoLabel) +
		styleMuted.Render(fmt.Sprintf("  %d worktrees", len(m.items)))
	if agentCount > 0 {
		header += styleMuted.Render(fmt.Sprintf("  ⎇ %d agents", agentCount))
	}
	if m.mux != tmux.None {
		header += styleMuted.Render("  " + m.mux.String())
	}
	b.WriteString(header + "\n\n")

	vis := m.visibleItems()

	// Dynamic column widths
	nameW := 18
	branchW := 22
	for _, it := range vis {
		if len(it.Name) > nameW-2 {
			nameW = min(len(it.Name)+2, 28)
		}
		if len(it.Branch) > branchW-2 {
			branchW = min(len(it.Branch)+2, 32)
		}
	}

	var listContent strings.Builder

	// Column headers
	colHeader := fmt.Sprintf(" %-3s%-*s %-*s %-8s %-4s %-8s %-6s %s",
		"", nameW, styleColHeader.Render("WORKTREE"),
		branchW, styleColHeader.Render("BRANCH"),
		styleColHeader.Render("STATUS"),
		styleColHeader.Render("CI"),
		styleColHeader.Render("AGENT"),
		styleColHeader.Render("REMOTE"),
		styleColHeader.Render("AGE"))
	listContent.WriteString(colHeader + "\n")

	for i, it := range vis {
		listContent.WriteString(m.viewRow(i, it, nameW, branchW) + "\n")
	}

	if len(vis) == 0 {
		listContent.WriteString(styleMuted.Render("  No worktrees found.\n"))
	}

	// Layout: list + optional detail panel
	listStr := listContent.String()
	if m.showDetail && len(vis) > 0 && m.cursor < len(vis) {
		detail := m.viewDetail(vis[m.cursor])
		detailWidth := min(44, m.width/3)
		detailStyled := styleDetailBorder.Width(detailWidth).Render(detail)
		listWidth := m.width - detailWidth - 4
		if listWidth > 20 {
			listStr = lipgloss.NewStyle().Width(listWidth).Render(listStr)
		}
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, listStr, detailStyled))
	} else {
		b.WriteString(listStr)
	}

	// Note bar
	if len(vis) > 0 && m.cursor < len(vis) {
		it := vis[m.cursor]
		if it.Meta.Note != "" {
			note := it.Meta.Note
			if idx := strings.Index(note, "\n"); idx > 0 {
				note = note[:idx] + "..."
			}
			if len(note) > m.width-6 && m.width > 10 {
				note = note[:m.width-9] + "..."
			}
			b.WriteString("\n" + styleNote.Render(" "+it.Name+" — "+note))
		}
	}

	// Status
	if m.statusMsg != "" {
		b.WriteString("\n" + styleMuted.Render(" "+m.statusMsg))
	}

	// Filter
	if m.mode == modeFilter {
		b.WriteString("\n " + m.filterInput.View())
	} else if m.filterText != "" {
		b.WriteString("\n" + styleMuted.Render(fmt.Sprintf(" filter: %s (/ to change, esc to clear)", m.filterText)))
	}

	// Help bar
	help := " a add  d del  D del+branch  n note  c color  p prune  / filter"
	if m.mux != tmux.None {
		help += "  t " + m.mux.String()
	}
	help += "  w ci  ? help  q quit"
	b.WriteString("\n\n" + styleHelp.Render(help))

	return b.String()
}

func (m Model) viewRow(idx int, it item, nameW, branchW int) string {
	// Cursor
	cursor := "  "
	nameStyle := lipgloss.NewStyle()
	if idx == m.cursor {
		cursor = styleCursor.Render("› ")
		nameStyle = styleCursor
	}
	if it.IsCurrent {
		cursor = styleCurrent.Render("● ")
		if idx == m.cursor {
			cursor = styleCurrent.Render("›●")
		}
		nameStyle = styleCurrent
	}

	// Color dot + icon
	dot := colorDot(it.Meta.Color)
	icon := it.Meta.Icon
	if icon == "" {
		icon = " "
	}

	name := nameStyle.Render(truncate(it.Name, nameW-1))

	// Branch
	branch := truncate(it.Branch, branchW-1)
	if it.IsDetached {
		branch = styleMuted.Render("(detached)")
	}

	// Status
	var status string
	if it.Status.Clean() {
		status = styleSuccess.Render("✓")
	} else {
		status = styleWarning.Render(it.Status.String())
	}

	// CI
	ciStr := ciIndicator(it.CI.State)

	// Agent
	agentStr := styleMuted.Render("—")
	if it.Agent != nil {
		if it.Agent.Active {
			agentStr = styleInfo.Render(it.Agent.Name)
		} else {
			agentStr = styleMuted.Render(it.Agent.Name)
		}
	}

	// Remote
	var remote string
	switch {
	case it.Remote.Gone:
		remote = styleError.Render("gone")
	case it.Remote.NoRemote:
		remote = styleMuted.Render("—")
	case it.Remote.Ahead == 0 && it.Remote.Behind == 0:
		remote = styleSuccess.Render("✓")
	default:
		remote = styleInfo.Render(it.Remote.String())
	}

	age := styleMuted.Render(it.Age)

	lock := " "
	if it.Locked {
		lock = "🔒"
	}

	return fmt.Sprintf("%s%s%s%-*s %-*s %-8s %-4s %-8s %-6s %s %s",
		cursor, dot, icon,
		nameW, name,
		branchW, branch,
		status,
		ciStr,
		agentStr,
		remote,
		age,
		lock)
}

func (m Model) viewDetail(it item) string {
	var b strings.Builder

	title := colorDot(it.Meta.Color) + " " + it.Meta.Icon
	if it.Meta.Icon == "" {
		title = colorDot(it.Meta.Color)
	}
	b.WriteString(styleHeader.Render(title+" "+it.Name) + "\n\n")

	row := func(label, val string) {
		b.WriteString(styleDetailLabel.Render(label) + styleDetailVal.Render(val) + "\n")
	}

	row("branch", it.Branch)
	row("path", it.Path)
	if !it.Remote.NoRemote {
		row("remote", it.Remote.String())
	}
	if it.Locked {
		row("locked", "yes")
	}
	row("age", it.Age)

	// Note
	if it.Meta.Note != "" {
		b.WriteString("\n" + styleColHeader.Render("NOTE") + "\n")
		b.WriteString(styleMuted.Render(it.Meta.Note) + "\n")
	}

	// CI
	if it.CI.State != "" && it.CI.State != "none" {
		b.WriteString("\n" + styleColHeader.Render("CI/CD") + "\n")
		b.WriteString(ciIndicator(it.CI.State) + " " + ciLabel(it.CI.State))
		if it.CI.RunName != "" {
			b.WriteString(styleMuted.Render("  " + it.CI.RunName))
		}
		b.WriteString("\n")
		if it.CI.RunURL != "" {
			b.WriteString(styleMuted.Render("  w to open in browser") + "\n")
		}
	}

	// Agent
	if it.Agent != nil {
		b.WriteString("\n" + styleColHeader.Render("AGENT") + "\n")
		if it.Agent.Active {
			b.WriteString(styleInfo.Render("● "+it.Agent.Name) + styleMuted.Render(" (active)") + "\n")
		} else {
			b.WriteString(styleMuted.Render("○ "+it.Agent.Name+" (inactive)") + "\n")
		}
	}

	// Changed files
	if len(it.Files) > 0 {
		b.WriteString("\n" + styleColHeader.Render("CHANGED FILES") + "\n")
		limit := min(len(it.Files), 10)
		for _, f := range it.Files[:limit] {
			b.WriteString(styleMuted.Render(f) + "\n")
		}
		if len(it.Files) > 10 {
			b.WriteString(styleMuted.Render(fmt.Sprintf("  ...and %d more", len(it.Files)-10)) + "\n")
		}
	}

	// Last commit
	if it.LastSHA != "" {
		b.WriteString("\n" + styleColHeader.Render("LAST COMMIT") + "\n")
		b.WriteString(styleInfo.Render(it.LastSHA) + " " + styleMuted.Render(it.LastMsg) + "\n")
	}

	return b.String()
}

// Modals

func (m Model) viewWithModal(modal string) string {
	modalStyled := styleModal.Render(modal)
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		modalStyled,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("235")),
	)
}

func (m Model) viewAddModal() string {
	var b strings.Builder
	b.WriteString(styleHeader.Render("New Worktree") + "\n\n")
	b.WriteString("Branch:  " + m.addInput.View() + "\n")

	limit := min(len(m.addMatches), 8)
	if limit > 0 {
		b.WriteString(styleMuted.Render("         ─────────────────────────") + "\n")
		for i := 0; i < limit; i++ {
			prefix := "         "
			if i == m.addIdx {
				b.WriteString(prefix + styleCursor.Render(m.addMatches[i]) + "\n")
			} else {
				b.WriteString(prefix + styleMuted.Render(m.addMatches[i]) + "\n")
			}
		}
		if len(m.addMatches) > limit {
			b.WriteString(styleMuted.Render(fmt.Sprintf("         ...%d more", len(m.addMatches)-limit)) + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString("Base:    " + styleMuted.Render(m.baseBranch) + "\n")

	if len(m.cfg.Hooks.PostCreate) > 0 {
		b.WriteString("\n" + styleMuted.Render("Post-create hooks:") + "\n")
		for _, h := range m.cfg.Hooks.PostCreate {
			b.WriteString(styleMuted.Render("  → "+h) + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(styleHelp.Render("enter create  tab autocomplete  ↑↓ navigate  esc cancel"))

	return b.String()
}

func (m Model) viewDeleteModal() string {
	vis := m.visibleItems()
	if m.cursor >= len(vis) {
		return ""
	}
	it := vis[m.cursor]

	var b strings.Builder
	b.WriteString(styleHeader.Render("Delete Worktree") + "\n\n")
	b.WriteString(fmt.Sprintf("  %s  %s\n", styleWarning.Render(it.Name), styleMuted.Render(it.Branch)))

	if it.Agent != nil && it.Agent.Active {
		b.WriteString("\n" + styleError.Render("  ⚠ "+it.Agent.Name+" agent is active in this worktree") + "\n")
	}

	b.WriteString("\n")
	if m.delBranch {
		b.WriteString(styleError.Render("  ⚠ Will also delete local branch") + "\n\n")
	}

	if len(m.cfg.Hooks.PreDelete) > 0 {
		b.WriteString(styleMuted.Render("  Pre-delete hooks will run:") + "\n")
		for _, h := range m.cfg.Hooks.PreDelete {
			b.WriteString(styleMuted.Render("    → "+h) + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(styleHelp.Render("y confirm  n/esc cancel"))

	return b.String()
}

func (m Model) viewPruneModal() string {
	var b strings.Builder
	b.WriteString(styleHeader.Render("Cleanup") + "\n\n")

	if len(m.pruneItems) == 0 {
		b.WriteString(styleMuted.Render("  Nothing to prune.") + "\n")
		b.WriteString("\n" + styleHelp.Render("esc close"))
		return b.String()
	}

	var gone, merged []int
	for i, pi := range m.pruneItems {
		switch pi.reason {
		case "gone":
			gone = append(gone, i)
		case "merged":
			merged = append(merged, i)
		}
	}

	renderGroup := func(title string, indices []int) {
		if len(indices) == 0 {
			return
		}
		b.WriteString(styleColHeader.Render(title) + "\n")
		for _, idx := range indices {
			pi := m.pruneItems[idx]
			cursor := "  "
			if idx == m.pruneCursor {
				cursor = styleCursor.Render("› ")
			}
			check := "☐"
			if pi.selected {
				check = styleSuccess.Render("☑")
			}

			name := pi.item.Name
			branch := styleMuted.Render(pi.item.Branch)
			age := styleMuted.Render(pi.item.Age)

			agentWarn := ""
			if pi.item.Agent != nil && pi.item.Agent.Active {
				agentWarn = styleError.Render(" ⚠ agent")
			}

			b.WriteString(fmt.Sprintf("%s%s %s  %s  %s%s\n", cursor, check, name, branch, age, agentWarn))
		}
		b.WriteString("\n")
	}

	renderGroup("⚠ Stale (remote branch deleted)", gone)
	renderGroup("✓ Merged (branch merged to "+m.baseBranch+")", merged)

	selected := 0
	for _, pi := range m.pruneItems {
		if pi.selected {
			selected++
		}
	}

	b.WriteString(styleHelp.Render(fmt.Sprintf(
		"space toggle  a all  enter prune (%d selected)  esc cancel", selected)))

	return b.String()
}

func (m Model) viewColorModal() string {
	vis := m.visibleItems()
	if m.cursor >= len(vis) {
		return ""
	}
	it := vis[m.cursor]

	var b strings.Builder
	b.WriteString(styleHeader.Render("Appearance: "+it.Name) + "\n\n")

	colorLabel := "  Color:  "
	if m.colorRow == 0 {
		colorLabel = styleCursor.Render("› Color:  ")
	}
	b.WriteString(colorLabel)
	for i, c := range colorPalette {
		dot := c.Dot
		if i == m.colorIdx {
			dot = lipgloss.NewStyle().
				Background(lipgloss.Color("99")).
				Render(dot)
		}
		b.WriteString(dot + " ")
	}
	b.WriteString("\n\n")

	iconLabel := "  Icon:   "
	if m.colorRow == 1 {
		iconLabel = styleCursor.Render("› Icon:   ")
	}
	b.WriteString(iconLabel)
	for i, ic := range iconPalette {
		s := ic
		if i == m.iconIdx {
			s = lipgloss.NewStyle().
				Background(lipgloss.Color("99")).
				Render(s)
		}
		b.WriteString(s + " ")
	}
	b.WriteString("\n\n")

	b.WriteString(styleHelp.Render("tab switch row  ←→ select  enter apply  esc cancel"))

	return b.String()
}

func (m Model) viewNoteModal() string {
	vis := m.visibleItems()
	if m.cursor >= len(vis) {
		return ""
	}
	it := vis[m.cursor]

	var b strings.Builder
	b.WriteString(styleHeader.Render("Note: "+it.Name) + "\n\n")
	b.WriteString(m.noteInput.View() + "\n\n")
	b.WriteString(styleHelp.Render("ctrl+s save  esc cancel"))

	return b.String()
}

func (m Model) viewTmuxModal() string {
	vis := m.visibleItems()
	if m.cursor >= len(vis) {
		return ""
	}
	it := vis[m.cursor]

	var b strings.Builder
	muxName := m.mux.String()
	b.WriteString(styleHeader.Render(muxName+": "+it.Name) + "\n\n")

	options := []string{
		"Open in new window",
		"Open in horizontal split",
		"Open in vertical split",
	}

	sessionName := m.cfg.Tmux.SessionPrefix + ":" + it.Name
	if tmux.SessionExists(sessionName) {
		options = append(options, "Kill session: "+sessionName)
	}

	for i, opt := range options {
		if i == m.tmuxCursor {
			b.WriteString(styleCursor.Render("  › "+opt) + "\n")
		} else {
			b.WriteString(styleMuted.Render("    "+opt) + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(styleHelp.Render("enter select  esc cancel"))

	return b.String()
}

func (m Model) viewHelp() string {
	var b strings.Builder

	b.WriteString(styleHeader.Render(" grove — git worktree manager") + "\n\n")

	section := func(title string, bindings [][2]string) {
		b.WriteString(styleColHeader.Render("  "+title) + "\n")
		for _, bind := range bindings {
			key := lipgloss.NewStyle().Width(14).Foreground(colorAccent).Render("  " + bind[0])
			b.WriteString(key + styleMuted.Render(bind[1]) + "\n")
		}
		b.WriteString("\n")
	}

	section("NAVIGATION", [][2]string{
		{"j/k ↑/↓", "move cursor"},
		{"g/G", "top / bottom"},
		{"l / Tab", "toggle detail panel"},
		{"h", "close detail panel"},
		{"/", "filter worktrees"},
		{"s", "cycle sort"},
	})

	section("ACTIONS", [][2]string{
		{"a", "add worktree"},
		{"d", "delete worktree"},
		{"D", "delete worktree + branch"},
		{"n", "edit note"},
		{"c", "color / icon picker"},
		{"p", "prune stale & merged"},
		{"r", "refresh"},
		{"enter", "open (cd into worktree)"},
	})

	section("INTEGRATIONS", [][2]string{
		{"t", "tmux / zellij menu"},
		{"w", "open CI run in browser"},
	})

	section("GENERAL", [][2]string{
		{"?", "toggle help"},
		{"q", "quit"},
		{"ctrl+c", "force quit"},
	})

	b.WriteString(styleHelp.Render("  Press any key to close help."))

	return b.String()
}

// Helpers

func ciIndicator(state string) string {
	switch state {
	case "passed":
		return styleSuccess.Render("✅")
	case "failed":
		return styleError.Render("❌")
	case "running":
		return styleWarning.Render("🔄")
	case "cancelled":
		return styleMuted.Render("⊘")
	default:
		return styleMuted.Render("—")
	}
}

func ciLabel(state string) string {
	switch state {
	case "passed":
		return styleSuccess.Render("passed")
	case "failed":
		return styleError.Render("failed")
	case "running":
		return styleWarning.Render("running")
	case "cancelled":
		return styleMuted.Render("cancelled")
	default:
		return styleMuted.Render("none")
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-1] + "…"
}
