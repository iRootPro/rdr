package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Theme is a flat 12-slot color palette. Swapping a Theme triggers
// a rebuild of every registered style so the running UI picks up the
// new palette without a restart. Slots map to the semantic roles the
// rest of the UI uses directly (background, foreground, accents,
// status colors).
type Theme struct {
	Name string
	Light bool // true for light palettes — drives glamour style etc.

	BG     lipgloss.Color // primary pane background
	AltBG  lipgloss.Color // selection / alternate row
	Border lipgloss.Color // muted border / separator
	Muted  lipgloss.Color // de-emphasized text (metadata, hints)
	Text   lipgloss.Color // default foreground

	Accent    lipgloss.Color // pane titles, active border, chrome
	Secondary lipgloss.Color // selected row, secondary accent

	Green  lipgloss.Color // counter / success
	Orange lipgloss.Color // time-ago / notice
	Red    lipgloss.Color // error
	Yellow lipgloss.Color // unread marker / warning
	Teal   lipgloss.Color // URLs / teal accent
}

// themeDark is the original TokyoNight-inspired palette. Kept 1:1 with
// the hardcoded values the UI shipped with before themes were added.
var themeDark = Theme{
	Name:      "dark",
	BG:        lipgloss.Color("#1a1b26"),
	AltBG:     lipgloss.Color("#24283b"),
	Border:    lipgloss.Color("#3b4261"),
	Muted:     lipgloss.Color("#565f89"),
	Text:      lipgloss.Color("#c0caf5"),
	Accent:    lipgloss.Color("#7aa2f7"),
	Secondary: lipgloss.Color("#bb9af7"),
	Green:     lipgloss.Color("#9ece6a"),
	Orange:    lipgloss.Color("#ff9e64"),
	Red:       lipgloss.Color("#f7768e"),
	Yellow:    lipgloss.Color("#e0af68"),
	Teal:      lipgloss.Color("#2ac3de"),
}

// themeLight uses Catppuccin Latte — a well-tested warm light palette
// that stays coherent with the Mocha theme below.
var themeLight = Theme{
	Name:      "light",
	Light:     true,
	BG:        lipgloss.Color("#eff1f5"),
	AltBG:     lipgloss.Color("#bcc0cc"),
	Border:    lipgloss.Color("#9ca0b0"),
	Muted:     lipgloss.Color("#7c7f93"),
	Text:      lipgloss.Color("#4c4f69"),
	Accent:    lipgloss.Color("#1e66f5"),
	Secondary: lipgloss.Color("#7533d4"),
	Green:     lipgloss.Color("#2e8b1e"),
	Orange:    lipgloss.Color("#e5590a"),
	Red:       lipgloss.Color("#c4102b"),
	Yellow:    lipgloss.Color("#a06208"),
	Teal:      lipgloss.Color("#0e7a80"),
}

// themeCatppuccinMocha — the canonical dark Catppuccin palette.
var themeCatppuccinMocha = Theme{
	Name:      "catppuccin",
	BG:        lipgloss.Color("#1e1e2e"),
	AltBG:     lipgloss.Color("#313244"),
	Border:    lipgloss.Color("#45475a"),
	Muted:     lipgloss.Color("#6c7086"),
	Text:      lipgloss.Color("#cdd6f4"),
	Accent:    lipgloss.Color("#89b4fa"),
	Secondary: lipgloss.Color("#cba6f7"),
	Green:     lipgloss.Color("#a6e3a1"),
	Orange:    lipgloss.Color("#fab387"),
	Red:       lipgloss.Color("#f38ba8"),
	Yellow:    lipgloss.Color("#f9e2af"),
	Teal:      lipgloss.Color("#94e2d5"),
}

// themeRosePine — Rose Pine main variant. The palette is intentionally
// sparse so several semantic slots reuse the same base color (Foam for
// both Green and Teal; Gold for both Orange and Yellow).
var themeRosePine = Theme{
	Name:      "rose-pine",
	BG:        lipgloss.Color("#191724"),
	AltBG:     lipgloss.Color("#1f1d2e"),
	Border:    lipgloss.Color("#26233a"),
	Muted:     lipgloss.Color("#6e6a86"),
	Text:      lipgloss.Color("#e0def4"),
	Accent:    lipgloss.Color("#ebbcba"),
	Secondary: lipgloss.Color("#c4a7e7"),
	Green:     lipgloss.Color("#9ccfd8"),
	Orange:    lipgloss.Color("#f6c177"),
	Red:       lipgloss.Color("#eb6f92"),
	Yellow:    lipgloss.Color("#f6c177"),
	Teal:      lipgloss.Color("#9ccfd8"),
}

// availableThemes is the cycle order exposed to the user through the
// Settings → General → Theme row. Index 0 is the default.
var availableThemes = []Theme{themeDark, themeLight, themeCatppuccinMocha, themeRosePine}

// Current color slots. Initial values match themeDark so the first
// paint works even before ApplyTheme is called. applyTheme reassigns
// each of these when the user switches palettes.
var (
	colorBG        = themeDark.BG
	colorAltBG     = themeDark.AltBG
	colorBorder    = themeDark.Border
	colorMuted     = themeDark.Muted
	colorText      = themeDark.Text
	colorAccent    = themeDark.Accent
	colorSecondary = themeDark.Secondary
	colorGreen     = themeDark.Green
	colorOrange    = themeDark.Orange
	colorRed       = themeDark.Red
	colorYellow    = themeDark.Yellow
	colorTeal      = themeDark.Teal

	glamourStyle = "dark"
)

// Style vars used across the main model views. Declared without values
// and populated by rebuildCoreStyles() so applyTheme can rebuild them
// with fresh colors at runtime.
var (
	paneActive           lipgloss.Style
	paneInactive         lipgloss.Style
	selectedRow          lipgloss.Style
	toastStyle           lipgloss.Style
	paneTitle            lipgloss.Style
	itemSelected         lipgloss.Style
	itemSelectedInactive lipgloss.Style
	unreadStyle          lipgloss.Style
	readStyle            lipgloss.Style
	counterStyle         lipgloss.Style
	timeAgoStyle         lipgloss.Style
	errStyle             lipgloss.Style
	statusBar            lipgloss.Style
)

// styleRebuilders holds the per-file rebuild functions registered via
// registerStyleRebuild. applyTheme iterates them after swapping colors
// so every cached Style in the package picks up the new palette.
var styleRebuilders []func()

func registerStyleRebuild(f func()) {
	styleRebuilders = append(styleRebuilders, f)
}

// ApplyTheme resolves a theme by name and applies it. Unknown names
// fall back to themeDark. Call this from main.go before ui.New and
// from the Settings screen when the user picks a new theme.
func ApplyTheme(name string) {
	applyTheme(themeByName(name))
}

func themeByName(name string) Theme {
	for _, t := range availableThemes {
		if t.Name == name {
			return t
		}
	}
	return themeDark
}

func applyTheme(t Theme) {
	colorBG = t.BG
	colorAltBG = t.AltBG
	colorBorder = t.Border
	colorMuted = t.Muted
	colorText = t.Text
	colorAccent = t.Accent
	colorSecondary = t.Secondary
	colorGreen = t.Green
	colorOrange = t.Orange
	colorRed = t.Red
	colorYellow = t.Yellow
	colorTeal = t.Teal
	if t.Light {
		glamourStyle = "light"
	} else {
		glamourStyle = "dark"
	}
	for _, cb := range styleRebuilders {
		cb()
	}
}

func init() {
	rebuildCoreStyles()
	registerStyleRebuild(rebuildCoreStyles)
}

func rebuildCoreStyles() {
	// Panes fill their full rectangle with the theme BG so the
	// terminal's own background never bleeds through. Without this,
	// light themes look dark because lipgloss only paints cells that
	// contain text.
	paneActive = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		BorderBackground(colorBG).
		Background(colorBG).
		Foreground(colorText).
		Padding(0, 1)

	paneInactive = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		BorderBackground(colorBG).
		Background(colorBG).
		Foreground(colorText).
		Padding(0, 1)

	selectedRow = lipgloss.NewStyle().Background(colorAltBG)

	toastStyle = lipgloss.NewStyle().
		Background(colorAltBG).
		Foreground(colorAccent).
		Bold(true)

	paneTitle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Background(colorBG).
		Bold(true).
		Padding(0, 0, 1, 0)

	itemSelected = lipgloss.NewStyle().
		Foreground(colorSecondary).
		Background(colorBG).
		Bold(true)

	itemSelectedInactive = lipgloss.NewStyle().
		Foreground(colorMuted).
		Background(colorBG)

	unreadStyle = lipgloss.NewStyle().Foreground(colorYellow).Background(colorBG)
	readStyle = lipgloss.NewStyle().Foreground(colorMuted).Background(colorBG)

	counterStyle = lipgloss.NewStyle().Foreground(colorGreen).Background(colorBG)
	timeAgoStyle = lipgloss.NewStyle().Foreground(colorOrange).Background(colorBG)
	errStyle = lipgloss.NewStyle().Foreground(colorRed).Background(colorBG)

	statusBar = lipgloss.NewStyle().
		Foreground(colorText).
		Background(colorBG).
		Padding(0, 1)
}

// paintLineBG takes a single rendered line (may contain ANSI codes) and
// ensures the theme background is active throughout: it replaces every
// ANSI full-reset with reset+bg-set and pads the line to the target
// width. Use this for status bars, help bars, breadcrumbs — anything
// that is assembled from multiple styled spans and displayed outside a
// pane wrapper.
func paintLineBG(line string, width int) string {
	restore := colorANSIFromHex(string(colorBG), 48) +
		colorANSIFromHex(string(colorText), 38)
	line = strings.ReplaceAll(line, "\x1b[0m", "\x1b[0m"+restore)
	line = restore + line
	if w := lipgloss.Width(line); w < width {
		line += lipgloss.NewStyle().Background(colorBG).
			Render(strings.Repeat(" ", width-w))
	}
	return line
}

// colorANSIFromHex builds a 24-bit ANSI color sequence. base is 48 for
// background or 38 for foreground.
func colorANSIFromHex(hex string, base int) string {
	if len(hex) != 7 || hex[0] != '#' {
		return ""
	}
	r := hexNibble(hex[1])*16 + hexNibble(hex[2])
	g := hexNibble(hex[3])*16 + hexNibble(hex[4])
	b := hexNibble(hex[5])*16 + hexNibble(hex[6])
	return fmt.Sprintf("\x1b[%d;2;%d;%d;%dm", base, r, g, b)
}

func hexNibble(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c - 'a' + 10)
	case c >= 'A' && c <= 'F':
		return int(c - 'A' + 10)
	}
	return 0
}
