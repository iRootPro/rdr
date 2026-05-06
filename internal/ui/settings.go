package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/iRootPro/rdr/internal/db"
	"github.com/iRootPro/rdr/internal/i18n"
)

var (
	settingsTitle       lipgloss.Style
	settingsKeyHint     lipgloss.Style
	settingsURL         lipgloss.Style
	settingsTabActive   lipgloss.Style
	settingsTabInactive lipgloss.Style
	settingsRowLabel    lipgloss.Style
	settingsRowValue    lipgloss.Style
)

func init() {
	rebuildSettingsStyles()
	registerStyleRebuild(rebuildSettingsStyles)
}

func rebuildSettingsStyles() {
	settingsTitle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Background(colorBG).
		Bold(true).
		Padding(0, 0, 1, 0)

	settingsKeyHint = lipgloss.NewStyle().
		Foreground(colorMuted).
		Background(colorBG).
		Italic(true)

	settingsURL = lipgloss.NewStyle().
		Foreground(colorTeal).
		Background(colorBG)

	settingsTabActive = lipgloss.NewStyle().
		Foreground(colorAccent).
		Background(colorBG).
		Bold(true)

	settingsTabInactive = lipgloss.NewStyle().
		Foreground(colorMuted).
		Background(colorBG)

	settingsRowLabel = lipgloss.NewStyle().
		Foreground(colorMuted).
		Background(colorBG)

	settingsRowValue = lipgloss.NewStyle().
		Foreground(colorText).
		Background(colorBG)
}

// generalRow describes one row in the flat General settings list. Label
// is the fixed left column (e.g. "Language"); Display is the
// currently-chosen value rendered in the right column (e.g. "English").
type generalRow struct {
	Label   string
	Display string
}

func buildGeneralRows(m *Model) []generalRow {
	mins := int(m.refreshInterval / time.Minute)
	refreshDisplay := m.tr.Settings.RefreshOff
	if mins > 0 {
		refreshDisplay = fmt.Sprintf(m.tr.Settings.RefreshFmt, mins)
	}
	retentionDisplay := m.tr.Settings.RetentionOff
	if days, _ := m.db.GetReadRetentionDays(); days > 0 {
		retentionDisplay = fmt.Sprintf(m.tr.Settings.RetentionFmt, days)
	}
	return []generalRow{
		{m.tr.Settings.LanguageLabel, langDisplayName(m.lang)},
		{m.tr.Settings.ImagesLabel, boolOnOff(m.showImages, m.tr)},
		{m.tr.Settings.SortLabel, sortDisplayName(m.sortField, m.sortReverse, m.tr)},
		{m.tr.Settings.PreviewLabel, boolOnOff(m.showPreview, m.tr)},
		{m.tr.Settings.ThemeLabel, m.themeName},
		{m.tr.Settings.RefreshLabel, refreshDisplay},
		{m.tr.Settings.RetentionLabel, retentionDisplay},
	}
}

func boolOnOff(v bool, tr *i18n.Strings) string {
	if v {
		return tr.Common.On
	}
	return tr.Common.Off
}

// sortDisplayName maps (sortField, sortReverse) to one of the four
// localized labels in tr.Sort. Unknown field falls back to date↓ so the
// UI never shows an empty cell.
func sortDisplayName(field string, reverse bool, tr *i18n.Strings) string {
	switch {
	case field == "date" && !reverse:
		return tr.Sort.DateDesc
	case field == "date" && reverse:
		return tr.Sort.DateAsc
	case field == "title" && !reverse:
		return tr.Sort.TitleAsc
	case field == "title" && reverse:
		return tr.Sort.TitleDesc
	}
	return tr.Sort.DateDesc
}

func renderSettings(m *Model, input string, width, height int) string {
	var b strings.Builder
	b.WriteString(settingsTitle.Render(m.tr.Settings.Title))
	b.WriteString("\n")

	b.WriteString(renderSettingsTabs(m.tr, m.settingsSection))
	b.WriteString("\n\n")

	switch m.settingsSection {
	case secGeneral:
		b.WriteString(renderGeneralSection(m))
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render(m.tr.Settings.GeneralHint))
	case secFolders:
		renderFoldersSection(&b, m, input)
	case secSmartFolders:
		renderSmartFoldersSection(&b, m, input)
	case secAfterSync:
		renderAfterSyncSection(&b, m, input)
	case secAI:
		renderAISection(&b, m, input)
	default:
		renderFeedsSection(&b, m, input)
	}

	contentW := width - 4
	if contentW < 20 {
		contentW = 20
	}
	renderW := contentW
	if renderW > 140 {
		renderW = 140
	}
	content := fillBackground(b.String(), renderW)
	if contentW > renderW {
		content = indentBlock(content, (contentW-renderW)/2)
		content = fillBackground(content, contentW)
	}
	return paneActive.Width(width - 2).Height(height - 2).Render(content)
}

func indentBlock(content string, pad int) string {
	if pad <= 0 {
		return content
	}
	prefix := strings.Repeat(" ", pad)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func renderSettingsTabs(tr *i18n.Strings, active settingsSection) string {
	tabs := []struct {
		label string
		sec   settingsSection
	}{
		{tr.Settings.SectionFeeds, secFeeds},
		{tr.Settings.SectionGeneral, secGeneral},
		{tr.Settings.SectionFolders, secFolders},
		{tr.Settings.SectionSmartFolders, secSmartFolders},
		{tr.Settings.SectionAfterSync, secAfterSync},
		{tr.Settings.SectionAI, secAI},
	}
	sepStyle := lipgloss.NewStyle().Background(colorBG)
	activeStyle := lipgloss.NewStyle().Foreground(colorBG).Background(colorAccent).Bold(true)
	inactiveStyle := lipgloss.NewStyle().Foreground(colorMuted).Background(colorBG)
	var cells []string
	for _, t := range tabs {
		if t.sec == active {
			cells = append(cells, activeStyle.Render(" "+t.label+" "))
		} else {
			cells = append(cells, inactiveStyle.Render(" "+t.label+" "))
		}
	}
	return strings.Join(cells, sepStyle.Render("  "))
}

func renderFeedsSection(b *strings.Builder, m *Model, input string) {
	tr := m.tr
	switch m.settingsMode {
	case smAddName:
		b.WriteString(tr.Settings.NewFeedName + "\n\n")
		b.WriteString(input)
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render(tr.Settings.EnterContinue))
		return
	case smAddURL:
		b.WriteString(tr.Settings.NewFeedURL + "\n\n")
		b.WriteString(input)
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render(tr.Settings.EnterContinue))
		return
	case smAddResolving:
		b.WriteString(tr.Settings.ResolvingYouTube + "\n\n")
		b.WriteString(settingsURL.Render(m.pendingURL))
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render(tr.Status.Fetching + " · esc cancel"))
		return
	case smAddCategoryPicker:
		renderCategoryPicker(b, m)
		return
	case smRename:
		b.WriteString(tr.Settings.RenameFeed + "\n\n")
		b.WriteString(input)
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render(tr.Settings.EnterSave))
		return
	case smCategory:
		b.WriteString(tr.Settings.CategoryPrompt + "\n\n")
		b.WriteString(input)
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render(tr.Settings.EnterSaveOrEmpty))
		return
	case smCategoryPicker:
		renderCategoryPicker(b, m)
		return
	case smImport:
		b.WriteString(tr.Settings.ImportPrompt + "\n\n")
		b.WriteString(input)
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render(tr.Settings.EnterSave))
		return
	case smExport:
		b.WriteString(tr.Settings.ExportPrompt + "\n\n")
		b.WriteString(input)
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render(tr.Settings.EnterSave))
		return
	}

	if len(m.feeds) == 0 {
		b.WriteString(readStyle.Render(tr.Settings.NoFeeds))
	} else {
		contentW := m.width - 4
		if contentW > 140 {
			contentW = 140
		}
		if contentW < 30 {
			contentW = 30
		}
		renderSettingsFeedRows(b, m, contentW)
	}
	b.WriteString("\n")
	b.WriteString(settingsKeyHint.Render(tr.Settings.FeedsHint))
}

func renderSettingsFeedRows(b *strings.Builder, m *Model, width int) {
	catStyle := lipgloss.NewStyle().Foreground(colorAccent).Background(colorBG).Bold(true)
	nameBase := lipgloss.NewStyle().Foreground(colorText).Background(colorBG)
	urlBase := settingsURL
	ordered := groupedSettingsFeeds(m.feeds)

	nameW := 0
	for _, item := range ordered {
		if item.feedIdx < 0 {
			continue
		}
		if w := lipgloss.Width(item.feed.Name); w > nameW {
			nameW = w
		}
	}
	if nameW < 14 {
		nameW = 14
	}
	if nameW > 28 {
		nameW = 28
	}
	prefixW := 4
	gapW := 2
	urlW := width - prefixW - nameW - gapW
	if urlW < 8 {
		urlW = 8
	}
	tableW := prefixW + nameW + gapW + urlW
	if tableW > width {
		tableW = width
	}

	for _, item := range ordered {
		if item.feedIdx < 0 {
			if item.category != "" && b.Len() > 0 {
				b.WriteString("\n")
			}
			label := "▸ " + strings.ToUpper(item.category)
			b.WriteString(catStyle.Render(label))
			b.WriteString("\n")
			continue
		}

		f := item.feed
		rowBG := colorBG
		prefix := "    "
		nameStyle := nameBase
		urlStyle := urlBase
		if item.feedIdx == m.settingsSel {
			rowBG = colorAltBG
			prefix = "  ▌ "
			nameStyle = itemSelected.Background(rowBG)
			urlStyle = settingsURL.Background(rowBG)
		}
		prefixRendered := lipgloss.NewStyle().Foreground(colorAccent).Background(rowBG).Render(prefix)
		nameCell := lipgloss.NewStyle().Width(nameW).MaxWidth(nameW).Background(rowBG).
			Render(nameStyle.Render(truncate(f.Name, nameW)))
		urlText := displaySettingsURL(f.URL)
		urlCell := lipgloss.NewStyle().Width(urlW).MaxWidth(urlW).Background(rowBG).
			Render(urlStyle.Render(truncate(urlText, urlW)))
		line := lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Background(rowBG).Render(prefixRendered),
			nameCell,
			lipgloss.NewStyle().Width(gapW).Background(rowBG).Render("  "),
			urlCell,
		)
		if item.feedIdx == m.settingsSel {
			line = lipgloss.NewStyle().Width(tableW).MaxWidth(tableW).Background(rowBG).Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
}

type settingsFeedRow struct {
	category string
	feedIdx  int
	feed     db.Feed
}

func groupedSettingsFeeds(feeds []db.Feed) []settingsFeedRow {
	type group struct {
		category string
		feeds    []settingsFeedRow
	}
	groupsByCat := make(map[string]*group)
	var order []string
	for i, f := range feeds {
		key := f.Category
		if _, ok := groupsByCat[key]; !ok {
			groupsByCat[key] = &group{category: displaySettingsCategory(key)}
			order = append(order, key)
		}
		groupsByCat[key].feeds = append(groupsByCat[key].feeds, settingsFeedRow{feedIdx: i, feed: f})
	}
	for i, key := range order {
		if key == "" {
			order = append(order[:i], order[i+1:]...)
			order = append(order, "")
			break
		}
	}

	out := make([]settingsFeedRow, 0, len(feeds)+len(order))
	for _, key := range order {
		g := groupsByCat[key]
		out = append(out, settingsFeedRow{category: g.category, feedIdx: -1})
		out = append(out, g.feeds...)
	}
	return out
}

func displaySettingsCategory(cat string) string {
	if strings.TrimSpace(cat) == "" {
		return "—"
	}
	return cat
}

func displaySettingsURL(raw string) string {
	return strings.TrimPrefix(strings.TrimPrefix(raw, "https://"), "http://")
}

// categoryPickerRow is one row in the folder picker opened by 'c' on a
// feed. Value is the category string stored on the feed; IsNew marks the
// synthetic "+ New folder…" row that routes to a text input instead of
// applying directly.
type categoryPickerRow struct {
	Name  string
	Value string
	IsNew bool
}

func buildCategoryPickerRows(m *Model) []categoryPickerRow {
	rows := []categoryPickerRow{
		{Name: m.tr.Settings.NoFolderOption, Value: ""},
	}
	for _, c := range allRegularFolders(m.feeds, m.regularFolders) {
		rows = append(rows, categoryPickerRow{Name: c, Value: c})
	}
	rows = append(rows, categoryPickerRow{
		Name:  m.tr.Settings.NewFolderOption,
		IsNew: true,
	})
	return rows
}

func renderCategoryPicker(b *strings.Builder, m *Model) {
	tr := m.tr
	rows := buildCategoryPickerRows(m)
	currentFolder := ""
	if m.settingsMode != smAddCategoryPicker && m.settingsSel < len(m.feeds) {
		currentFolder = m.feeds[m.settingsSel].Category
	}

	b.WriteString(settingsTitle.Render(tr.Settings.CategoryPickerTitle))
	b.WriteString("\n")

	checkStyle := lipgloss.NewStyle().Foreground(colorAccent).Background(colorBG)
	newRowStyle := lipgloss.NewStyle().Foreground(colorAccent).Background(colorBG).Italic(true)

	for i, r := range rows {
		prefix := "  "
		mark := "  "
		style := lipgloss.NewStyle().Foreground(colorText).Background(colorBG)
		if r.IsNew {
			style = newRowStyle
		}
		if i == m.settingsCategoryPickerSel {
			prefix = "› "
			style = itemSelected
		}
		if !r.IsNew && r.Value == currentFolder {
			mark = checkStyle.Render(" ✓")
		}
		b.WriteString(prefix + style.Render(r.Name) + mark)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(settingsKeyHint.Render(tr.Settings.CategoryPickerHint))
}

// renderFoldersSection draws regular folders with a per-folder feed count.
// Folders are persisted independently so they may be empty; feed.Category
// stores the assignment from feed to folder.
func renderFoldersSection(b *strings.Builder, m *Model, input string) {
	tr := m.tr
	if m.settingsMode == smFolderAdd {
		b.WriteString(tr.Settings.FolderAdd + "\n\n")
		b.WriteString(input)
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render(tr.Settings.EnterSave))
		return
	}
	if m.settingsMode == smFolderRename {
		b.WriteString(tr.Settings.FolderRename + "\n\n")
		b.WriteString(input)
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render(tr.Settings.EnterSaveOrEmpty))
		return
	}

	cats := allRegularFolders(m.feeds, m.regularFolders)
	if len(cats) == 0 {
		b.WriteString(readStyle.Render(tr.Settings.NoFolders))
	} else {
		counts := categoryCounts(m.feeds)
		nameW := 0
		for _, c := range cats {
			if w := lipgloss.Width(c); w > nameW {
				nameW = w
			}
		}
		nameCell := lipgloss.NewStyle().Width(nameW + 2)
		countStyle := lipgloss.NewStyle().Foreground(colorMuted).Background(colorBG)
		for i, c := range cats {
			prefix := "  "
			nameStyle := lipgloss.NewStyle().Foreground(colorAccent).Background(colorBG).Bold(true)
			if i == m.settingsFolderSel {
				prefix = "› "
				nameStyle = itemSelected
			}
			line := prefix + nameCell.Render(nameStyle.Render(c)) +
				countStyle.Render(fmt.Sprintf("(%d)", counts[c]))
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(settingsKeyHint.Render(tr.Settings.FoldersHint))
}

// renderSmartFoldersSection draws the smart folders list with name on
// the left and query on the right. Supports the same prompt flow as
// feeds (smSmartFolderAddName/AddQuery/EditName/EditQuery).
func renderSmartFoldersSection(b *strings.Builder, m *Model, input string) {
	tr := m.tr
	switch m.settingsMode {
	case smSmartFolderAddName:
		b.WriteString(tr.Settings.SmartFolderAddName + "\n\n")
		b.WriteString(input)
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render(tr.Settings.EnterContinue))
		return
	case smSmartFolderAddQuery:
		b.WriteString(tr.Settings.SmartFolderAddQuery + "\n\n")
		b.WriteString(input)
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render(tr.Settings.EnterSave))
		return
	case smSmartFolderEditName:
		b.WriteString(tr.Settings.SmartFolderEditName + "\n\n")
		b.WriteString(input)
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render(tr.Settings.EnterContinue))
		return
	case smSmartFolderEditQuery:
		b.WriteString(tr.Settings.SmartFolderEditQuery + "\n\n")
		b.WriteString(input)
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render(tr.Settings.EnterSave))
		return
	}

	if len(m.smartFolders) == 0 {
		b.WriteString(readStyle.Render(tr.Settings.NoSmartFolders))
	} else {
		// Fixed-width name column so queries line up under each other.
		nameW := 0
		for _, f := range m.smartFolders {
			if w := lipgloss.Width(f.Name); w > nameW {
				nameW = w
			}
		}
		nameCell := lipgloss.NewStyle().Width(nameW + 2)
		queryStyle := lipgloss.NewStyle().Foreground(colorMuted).Background(colorBG).Italic(true)
		for i, f := range m.smartFolders {
			prefix := "  "
			nameStyle := lipgloss.NewStyle().Foreground(colorAccent).Background(colorBG).Bold(true)
			if i == m.settingsSmartFolderSel {
				prefix = "› "
				nameStyle = itemSelected
			}
			line := prefix + nameCell.Render(nameStyle.Render(f.Name)) + queryStyle.Render(f.Query)
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(settingsKeyHint.Render(tr.Settings.SmartFoldersHint))
}

func renderGeneralSection(m *Model) string {
	rows := buildGeneralRows(m)

	// Fixed-width label column so values line up regardless of which row
	// is widest. Width is computed from the longest Label in the current
	// language.
	labelW := 0
	for _, r := range rows {
		if w := lipgloss.Width(r.Label); w > labelW {
			labelW = w
		}
	}

	var b strings.Builder
	for i, r := range rows {
		rowBG := colorBG
		if i == m.settingsGeneralSel {
			rowBG = colorAltBG
		}
		labelStyle := lipgloss.NewStyle().Foreground(colorMuted).Background(rowBG)
		valueStyle := lipgloss.NewStyle().Foreground(colorText).Background(rowBG)
		prefixStyle := lipgloss.NewStyle().Background(rowBG)
		cellStyle := lipgloss.NewStyle().Width(labelW + 2).Background(rowBG)

		prefix := "  "
		if i == m.settingsGeneralSel {
			prefix = "› "
			valueStyle = lipgloss.NewStyle().Foreground(colorSecondary).Background(rowBG).Bold(true)
		}

		label := labelStyle.Render(r.Label + ":")
		value := valueStyle.Render(r.Display)
		line := prefixStyle.Render(prefix) + cellStyle.Render(label) + value
		b.WriteString(line)
		if i < len(rows)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// renderAfterSyncSection draws the list of after-sync commands. Same
// add/edit/delete flow as smart folders.
func renderAfterSyncSection(b *strings.Builder, m *Model, input string) {
	tr := m.tr
	switch m.settingsMode {
	case smAfterSyncAdd:
		b.WriteString(tr.Settings.AfterSyncAdd + "\n\n")
		b.WriteString(input)
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render(tr.Settings.EnterSave))
		return
	case smAfterSyncEdit:
		b.WriteString(tr.Settings.AfterSyncEdit + "\n\n")
		b.WriteString(input)
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render(tr.Settings.EnterSave))
		return
	}

	if len(m.afterSyncCommands) == 0 {
		b.WriteString(readStyle.Render(tr.Settings.NoAfterSync))
	} else {
		for i, cmd := range m.afterSyncCommands {
			prefix := "  "
			style := lipgloss.NewStyle().Foreground(colorText).Background(colorBG)
			if i == m.settingsAfterSyncSel {
				prefix = "› "
				style = itemSelected
			}
			b.WriteString(prefix + style.Render(cmd))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(settingsKeyHint.Render(tr.Settings.AfterSyncHint))
}
