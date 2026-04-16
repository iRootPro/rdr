package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/iRootPro/rdr/internal/ai"
	"github.com/iRootPro/rdr/internal/db"
	"github.com/iRootPro/rdr/internal/i18n"
)

// loadAIConfig reads AI settings from the database.
func loadAIConfig(database *db.DB) ai.Config {
	provider, _ := database.GetAIProvider()
	endpoint, _ := database.GetAIEndpoint()
	apiKey, _ := database.GetAIKey()
	model, _ := database.GetAIModel()
	if provider == "" {
		provider = ai.ProviderOpenAI
	}
	return ai.Config{
		Provider: provider,
		Endpoint: endpoint,
		APIKey:   apiKey,
		Model:    model,
	}
}

// translateCmd creates a tea.Cmd that translates article text.
func translateCmd(cfg ai.Config, text, targetLang string) tea.Cmd {
	return func() tea.Msg {
		result, err := ai.Translate(context.Background(), cfg, text, targetLang)
		if err != nil {
			return aiErrorMsg{err}
		}
		return aiResultMsg{kind: "translate", content: result}
	}
}

// summarizeCmd creates a tea.Cmd that summarizes article text.
func summarizeCmd(cfg ai.Config, text, lang string) tea.Cmd {
	return func() tea.Msg {
		result, err := ai.Summarize(context.Background(), cfg, text, lang)
		if err != nil {
			return aiErrorMsg{err}
		}
		return aiResultMsg{kind: "summarize", content: result}
	}
}

// targetLang returns the translation target language name based on
// current UI language: if UI is Russian → translate to Russian,
// otherwise → translate to English.
func targetLang(lang i18n.Lang) string {
	if lang == i18n.RU {
		return "Russian"
	}
	return "English"
}

// langName returns human-readable language name for summarization.
func langName(lang i18n.Lang) string {
	if lang == i18n.RU {
		return "Russian"
	}
	return "English"
}

// articlePlainText extracts plain text from an article for AI processing.
func articlePlainText(a *db.Article) string {
	if a.CachedBody != "" {
		return stripHTML(a.CachedBody)
	}
	if a.Content != "" {
		return stripHTML(a.Content)
	}
	return stripHTML(a.Description)
}

// ── AI Settings Tab ──

// settingsAIRows returns label-value rows for the AI settings section.
type aiRow struct {
	Label string
	Value string
	Key   string // "endpoint", "key", "model"
}

func buildAIRows(m *Model) []aiRow {
	mask := m.aiConfig.APIKey
	if len(mask) > 4 {
		mask = "****" + mask[len(mask)-4:]
	}
	providerDisplay := m.aiConfig.Provider
	if providerDisplay == "" {
		providerDisplay = ai.ProviderOpenAI
	}
	return []aiRow{
		{m.tr.Settings.AIProviderLabel, providerDisplay, "provider"},
		{m.tr.Settings.AIEndpointLabel, m.aiConfig.Endpoint, "endpoint"},
		{m.tr.Settings.AIKeyLabel, mask, "key"},
		{m.tr.Settings.AIModelLabel, m.aiConfig.Model, "model"},
	}
}

// renderAISection draws the AI settings tab.
func renderAISection(b *strings.Builder, m *Model, input string) {
	tr := m.tr

	switch m.settingsMode {
	case smAIEdit:
		rows := buildAIRows(m)
		if m.settingsAISel < len(rows) {
			b.WriteString(rows[m.settingsAISel].Label + ":\n\n")
		}
		b.WriteString(input)
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render(tr.Settings.EnterSave))
		return
	}

	if !m.aiConfig.Enabled() {
		b.WriteString(readStyle.Render(tr.Settings.AINotConfigured))
		b.WriteString("\n\n")
	}

	rows := buildAIRows(m)
	labelW := 0
	for _, r := range rows {
		if w := lipgloss.Width(r.Label); w > labelW {
			labelW = w
		}
	}

	for i, r := range rows {
		prefix := "  "
		labelStyle := lipgloss.NewStyle().Foreground(colorMuted).Background(colorBG)
		valueStyle := lipgloss.NewStyle().Foreground(colorText).Background(colorBG)
		if i == m.settingsAISel {
			prefix = "› "
			valueStyle = itemSelected
		}
		val := r.Value
		if val == "" {
			val = "—"
		}
		label := labelStyle.Render(fmt.Sprintf("%-*s", labelW, r.Label))
		b.WriteString(prefix + label + "  " + valueStyle.Render(val))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(settingsKeyHint.Render(tr.Settings.AIHint))
}

// updateSettingsAI handles keystrokes in the AI settings tab.
func (m Model) updateSettingsAI(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	rows := buildAIRows(&m)
	switch {
	case key.Matches(msg, m.keys.Down):
		if m.settingsAISel < len(rows)-1 {
			m.settingsAISel++
		}
		return m, nil
	case key.Matches(msg, m.keys.Up):
		if m.settingsAISel > 0 {
			m.settingsAISel--
		}
		return m, nil
	case key.Matches(msg, m.keys.Enter), keyIs(msg, "e"):
		if m.settingsAISel >= len(rows) {
			return m, nil
		}
		// Provider toggles between openai/claude instead of text input.
		if rows[m.settingsAISel].Key == "provider" {
			if m.aiConfig.Provider == ai.ProviderClaude {
				m.aiConfig.Provider = ai.ProviderOpenAI
			} else {
				m.aiConfig.Provider = ai.ProviderClaude
			}
			_ = m.db.SetAIProvider(m.aiConfig.Provider)
			return m, nil
		}
		m.settingsMode = smAIEdit
		switch rows[m.settingsAISel].Key {
		case "endpoint":
			m.settingsInput.SetValue(m.aiConfig.Endpoint)
		case "key":
			m.settingsInput.SetValue(m.aiConfig.APIKey)
		case "model":
			m.settingsInput.SetValue(m.aiConfig.Model)
		}
		m.settingsInput.CursorEnd()
		m.settingsInput.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

// submitAIEdit saves the edited AI setting value.
func submitAIEdit(m *Model, value string) {
	rows := buildAIRows(m)
	if m.settingsAISel >= len(rows) {
		return
	}
	switch rows[m.settingsAISel].Key {
	case "provider":
		// provider is toggled via Enter, not edited via text
	case "endpoint":
		m.aiConfig.Endpoint = value
		_ = m.db.SetAIEndpoint(value)
	case "key":
		m.aiConfig.APIKey = value
		_ = m.db.SetAIKey(value)
	case "model":
		m.aiConfig.Model = value
		_ = m.db.SetAIModel(value)
	}
}
