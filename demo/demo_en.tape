# rdr — full demo (English)
Output demo/demo_en.gif

Set Shell zsh
Set FontFamily "JetBrainsMono Nerd Font Mono"
Set FontSize 13
Set Width 1600
Set Height 900
Set Padding 10
Set TypingSpeed 50ms

# Launch with English dev data
Type "RDR_HOME=./dev-en go run ."
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

# ── Move to a Habr article (has full content to fetch) ──
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

# Open and load full article
Enter
Sleep 1.5s
Type "f"
Sleep 5s

# Scroll loaded full article
Down
Sleep 150ms
Down
Sleep 150ms
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

# ── Star an article ──
Type "m"
Sleep 800ms
Down
Sleep 300ms
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

# General tab
Tab
Sleep 1s

# Folders tab
Tab
Sleep 1s

# Smart folders tab
Tab
Sleep 1s

# Auto-commands tab
Tab
Sleep 1s

# Close settings
Escape
Sleep 800ms

# ── Zen mode ──
Type "z"
Sleep 1.5s
Type "z"
Sleep 800ms

# ── Help ──
Type "?"
Sleep 2s
Escape
Sleep 800ms

# ── Quit ──
Type "q"
Sleep 500ms
