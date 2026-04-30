# Title

rdr — a terminal RSS reader in Go (Bubble Tea), with full-text reading and vim keys

# Body

Built **rdr** because I wanted an RSS reader that lives in the terminal and works over SSH.

- Split-pane TUI: feeds left, articles right, full-screen reader (readability + glamour)
- Vim navigation, query language (`title:rust unread newer:1w`), smart folders
- Inline images via Kitty Graphics Protocol
- Optional AI translate / summarize (Claude, Ollama, any OpenAI-compatible API)
- 4 themes, OPML import/export, OSC 52 clipboard

Install: `brew tap iRootPro/tap && brew install rdr` or `go install github.com/iRootPro/rdr@latest`

Repo: https://github.com/iRootPro/rdr

Feedback very welcome — especially on what's missing.

# Subreddits

- r/commandline
- r/golang
- r/linux

# Video to upload

`demo/using_reddit.mp4` (1920×1186, 5.8 MB, 50 sec)
