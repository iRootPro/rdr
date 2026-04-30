# rdr — demo matching user's Kitty setup (Kanagawa + Meslo Nerd Font + fish)
Output demo/demo_kitty.gif

Set Shell      "fish"
Set FontFamily "MesloLGLDZ Nerd Font Mono"
Set FontSize   14
Set Width      1600
Set Height     900
Set Padding    4
Set TypingSpeed 50ms
Set Theme      { "name": "Kanagawa", "background": "#1f1f28", "foreground": "#dcd7ba", "cursor": "#c8c093", "selection": "#2d4f67", "black": "#16161d", "brightBlack": "#727169", "red": "#c34043", "brightRed": "#e82424", "green": "#76946a", "brightGreen": "#98bb6c", "yellow": "#c0a36e", "brightYellow": "#e6c384", "blue": "#7e9cd8", "brightBlue": "#7fb4ca", "magenta": "#957fb8", "brightMagenta": "#938aa9", "cyan": "#6a9589", "brightCyan": "#7aa89f", "white": "#c8c093", "brightWhite": "#dcd7ba" }

# Launch real rdr (uses ~/.config/rdr)
Type "rdr"
Enter
Sleep 3s

# ── Feed list navigation ──
Sleep 1s
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 400ms
Up
Sleep 400ms
Up
Sleep 400ms

# Collapse/expand category
Space
Sleep 800ms
Space
Sleep 800ms

# ── Switch to articles ──
Tab
Sleep 1s

# Scroll articles
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms

# ── Open article in reader ──
Enter
Sleep 2s

# Scroll reader content
Down
Sleep 150ms
Down
Sleep 150ms
Down
Sleep 150ms
Down
Sleep 150ms

# Back to articles
Escape
Sleep 1s

# ── Star + unstar same article (no net change) ──
Type "m"
Sleep 800ms
Type "m"
Sleep 800ms

# ── Filter: unread only ──
Type "u"
Sleep 1s

# ── Filter: starred ──
Shift+S
Sleep 1s

# ── Filter: all ──
Type "a"
Sleep 1s

# ── Search ──
Type "/"
Sleep 500ms
Type "rust"
Sleep 1s
Escape
Sleep 800ms

# ── Settings ──
Type "s"
Sleep 1s
Tab
Sleep 1s
Tab
Sleep 1s
Tab
Sleep 1s
Tab
Sleep 1s
Escape
Sleep 800ms

# ── Help ──
Type "?"
Sleep 2s
Escape
Sleep 800ms

# ── Quit ──
Type "q"
Sleep 500ms
