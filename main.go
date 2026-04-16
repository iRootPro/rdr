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

	showImages, _ := database.GetShowImages()
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

	migrateSmartFolders(database, cfg)

	fetcher := feed.New(database)
	program := tea.NewProgram(
		ui.New(database, fetcher, cfg.AfterSyncCommands, cfg.RefreshInterval, home, lang, showImages, sortField, sortReverse, showPreview, themeName),
		tea.WithAltScreen(),
	)
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "run:", err)
		os.Exit(1)
	}
}
