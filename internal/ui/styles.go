package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorBG        = lipgloss.Color("#1a1b26")
	colorAltBG     = lipgloss.Color("#24283b")
	colorBorder    = lipgloss.Color("#3b4261")
	colorMuted     = lipgloss.Color("#565f89")
	colorText      = lipgloss.Color("#c0caf5")
	colorAccent    = lipgloss.Color("#7aa2f7")
	colorSecondary = lipgloss.Color("#bb9af7")
	colorGreen     = lipgloss.Color("#9ece6a")
	colorOrange    = lipgloss.Color("#ff9e64")
	colorRed       = lipgloss.Color("#f7768e")
	colorYellow    = lipgloss.Color("#e0af68")
	colorTeal      = lipgloss.Color("#2ac3de")

	paneActive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(0, 1)

	paneInactive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	// selectedRow paints a soft background across the full row width
	// behind the currently focused list item. Combined with itemSelected
	// on the text, the effect is a highlighted bar rather than a bare
	// colour shift.
	selectedRow = lipgloss.NewStyle().Background(colorAltBG)

	paneTitle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true).
			Padding(0, 0, 1, 0)

	itemSelected = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	itemSelectedInactive = lipgloss.NewStyle().
				Foreground(colorMuted)

	unreadStyle = lipgloss.NewStyle().Foreground(colorYellow)
	readStyle   = lipgloss.NewStyle().Foreground(colorMuted)

	counterStyle = lipgloss.NewStyle().Foreground(colorGreen)
	timeAgoStyle = lipgloss.NewStyle().Foreground(colorOrange)
	errStyle     = lipgloss.NewStyle().Foreground(colorRed)

	statusBar = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)
)
