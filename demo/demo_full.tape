# rdr — full feature showcase for Reddit
Output demo/demo_full.gif

Set Shell zsh
Set FontFamily "JetBrainsMono Nerd Font Mono"
Set FontSize 14
Set Width 1800
Set Height 1000
Set Padding 10
Set TypingSpeed 40ms

# Launch
Type "RDR_HOME=./dev go run ."
Enter
Sleep 3s

# ── Feed list with icons & categories ──
Sleep 1s
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms

# Collapse/expand
Space
Sleep 600ms
Space
Sleep 600ms

# ── Articles ──
Tab
Sleep 1s
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms

# ── Open article ──
Enter
Sleep 2s
Down
Sleep 100ms
Down
Sleep 100ms
Down
Sleep 100ms
Down
Sleep 100ms

# ── Fetch full article ──
Escape
Sleep 500ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Enter
Sleep 1s
Type "f"
Sleep 4s
Down
Sleep 100ms
Down
Sleep 100ms
Down
Sleep 100ms
Down
Sleep 100ms
Down
Sleep 100ms
Down
Sleep 100ms

# Back
Escape
Sleep 800ms

# ── Star ──
Type "m"
Sleep 500ms
Down
Sleep 200ms
Type "m"
Sleep 500ms

# ── Filters ──
Type "u"
Sleep 800ms
Shift+S
Sleep 800ms
Type "a"
Sleep 800ms

# ── Search ──
Type "/"
Sleep 400ms
Type "go"
Sleep 800ms
Escape
Sleep 600ms

# ── Command mode ──
Type ":"
Sleep 400ms
Type "sort title"
Enter
Sleep 800ms
Type ":"
Sleep 400ms
Type "sort date"
Enter
Sleep 800ms

# ── Settings ──
Type "s"
Sleep 800ms
Tab
Sleep 800ms
Tab
Sleep 800ms
Tab
Sleep 800ms
Tab
Sleep 800ms
Tab
Sleep 800ms
Escape
Sleep 600ms

# ── Zen mode ──
Type "z"
Sleep 1.5s
Type "z"
Sleep 600ms

# ── Discover ──
Type ":"
Sleep 300ms
Type "discover"
Enter
Sleep 1.5s
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms
Escape
Sleep 800ms

# ── Help ──
Type "?"
Sleep 2s
Escape
Sleep 600ms

# ── Quit ──
Type "q"
Sleep 500ms
