package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/iRootPro/rdr/internal/config"
	"github.com/iRootPro/rdr/internal/db"
)

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

	feeds, err := database.ListFeeds()
	if err != nil {
		log.Fatalf("list feeds: %v", err)
	}

	fmt.Printf("rdr: home=%s, %d feed(s)\n", home, len(feeds))
	for _, f := range feeds {
		fmt.Printf("  [%d] %s — %s (unread: %d)\n",
			f.Position, f.Name, f.URL, f.UnreadCount)
	}
}
