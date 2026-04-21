package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

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

	content := fillBackground(b.String(), width-4)
	return paneActive.Width(width - 2).Height(height - 2).Render(content)
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
	var cells []string
	for _, t := range tabs {
		if t.sec == active {
			cells = append(cells, settingsTabActive.Render("["+t.label+"]"))
		} else {
			cells = append(cells, settingsTabInactive.Render(" "+t.label+" "))
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
		b.WriteString(settingsKeyHint.Render(tr.Settings.EnterSave))
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
		catStyle := lipgloss.NewStyle().Foreground(colorAccent).Background(colorBG).Bold(true)
		lastCat := ""
		for i, f := range m.feeds {
			cat := f.Category
			if cat == "" {
				cat = "—"
			}
			if cat != lastCat {
				if lastCat != "" {
					b.WriteString("\n")
				}
				b.WriteString(catStyle.Render("▸ "+cat))
				b.WriteString("\n")
				lastCat = cat
			}
			prefix := "    "
			nameStyle := lipgloss.NewStyle().Foreground(colorText).Background(colorBG)
			if i == m.settingsSel {
				prefix = "  › "
				nameStyle = itemSelected
			}
			line := fmt.Sprintf("%s%s  %s",
				prefix,
				nameStyle.Render(f.Name),
				settingsURL.Render(f.URL),
			)
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(settingsKeyHint.Render(tr.Settings.FeedsHint))
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
	for _, c := range uniqueCategories(m.feeds) {
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
	if m.settingsSel < len(m.feeds) {
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

// renderFoldersSection draws the list of unique non-empty feed folders
// (= the Category column on feeds) with a per-folder feed count. The
// folder list is purely derived — there is no separate "folders" table.
// Supports smFolderRename prompt with "empty = Other" semantics.
func renderFoldersSection(b *strings.Builder, m *Model, input string) {
	tr := m.tr
	if m.settingsMode == smFolderRename {
		b.WriteString(tr.Settings.FolderRename + "\n\n")
		b.WriteString(input)
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render(tr.Settings.EnterSaveOrEmpty))
		return
	}

	cats := uniqueCategories(m.feeds)
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
