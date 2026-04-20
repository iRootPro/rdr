package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iRootPro/rdr/internal/config"
	"github.com/iRootPro/rdr/internal/db"
	"github.com/iRootPro/rdr/internal/feed"
	"github.com/iRootPro/rdr/internal/i18n"
	"github.com/iRootPro/rdr/internal/kitty"
	"github.com/iRootPro/rdr/internal/ui"
)

// migrateSmartFolders seeds the smart_folders table from cfg.SmartFolders
// on first run after upgrading. Guarded by a flag in the settings table so
// a user who deletes their folders through the UI doesn't see them come
// back on the next startup.
func migrateSmartFolders(database *db.DB, cfg *config.Config) {
	if len(cfg.SmartFolders) == 0 {
		return
	}
	migrated, _ := database.GetSetting("smart_folders_migrated")
	if migrated != "" {
		return
	}
	for _, f := range cfg.SmartFolders {
		if _, err := database.InsertSmartFolder(f.Name, f.Query); err != nil {
			log.Printf("migrate smart folder %q: %v", f.Name, err)
		}
	}
	_ = database.SetSetting("smart_folders_migrated", "true")
}

func main() {
	home, err := config.ResolveHome()
	if err != nil {
		log.Fatalf("resolve home: %v", err)
	}

	database, err := db.Open(filepath.Join(home, "rdr.db"))
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	cfg, err := config.Load(home)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := config.Sync(database, cfg); err != nil {
		log.Fatalf("sync feeds: %v", err)
	}

	langStr, _ := database.GetLanguage()
	lang := i18n.Parse(langStr)

	// show_images: if the setting has never been written (fresh user),
	// default ON when the terminal natively supports Kitty Graphics so
	// inline images "just work" instead of waiting for the user to
	// discover the `:images` toggle. Any explicit setting — including
	// "false" — is honoured.
	showImagesRaw, _ := database.GetSetting("show_images")
	var showImages bool
	if showImagesRaw == "" {
		showImages = kitty.IsSupported()
		_ = database.SetShowImages(showImages)
	} else {
		showImages = showImagesRaw == "true"
	}
	// tmux strips unknown APC sequences unless allow-passthrough is on.
	// Log a hint so users who wonder why images don't render have a
	// breadcrumb in the log file.
	if showImages && kitty.InsideTmux() {
		log.Printf("images: running inside tmux — add `set -g allow-passthrough on` to tmux.conf if inline images don't render")
	}
	sortField, _ := database.GetSortField()
	if sortField == "" {
		sortField = "date"
	}
	sortReverse, _ := database.GetSortReverse()
	showPreview, _ := database.GetShowPreview()
	themeName, _ := database.GetTheme()
	if themeName == "" {
		themeName = "dark"
	}
	ui.ApplyTheme(themeName)

	// Refresh interval: DB overrides config (DB = 0 means check config).
	refreshMinutes, _ := database.GetRefreshInterval()
	if refreshMinutes == 0 {
		refreshMinutes = cfg.RefreshInterval
	}
	// After-sync commands: merge DB + config (DB takes priority).
	afterSync, _ := database.GetAfterSyncCommands()
	if len(afterSync) == 0 {
		afterSync = cfg.AfterSyncCommands
	}

	migrateSmartFolders(database, cfg)

	fetcher := feed.New(database)
	program := tea.NewProgram(
		ui.New(database, fetcher, afterSync, refreshMinutes, home, lang, showImages, sortField, sortReverse, showPreview, themeName),
		tea.WithAltScreen(),
	)
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "run:", err)
		os.Exit(1)
	}
}
