package db

import "testing"

func TestCreateFolder_ListFoldersShowsEmptyFolder(t *testing.T) {
	d := openTestDB(t)

	if err := d.CreateFolder("YouTube"); err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	folders, err := d.ListFolders()
	if err != nil {
		t.Fatalf("ListFolders: %v", err)
	}
	if len(folders) != 1 || folders[0] != "YouTube" {
		t.Fatalf("folders = %#v, want [YouTube]", folders)
	}
}

func TestRenameFolder_RenamesExplicitFolderAndFeeds(t *testing.T) {
	d := openTestDB(t)
	f, _ := d.UpsertFeed("A", "https://a.example/rss", "YouTube")
	if err := d.CreateFolder("YouTube"); err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}

	if err := d.RenameFolder("YouTube", "Video"); err != nil {
		t.Fatalf("RenameFolder: %v", err)
	}
	feeds, _ := d.ListFeeds()
	if got := feedByID(feeds, f.ID).Category; got != "Video" {
		t.Fatalf("category = %q, want Video", got)
	}
	folders, _ := d.ListFolders()
	if len(folders) != 1 || folders[0] != "Video" {
		t.Fatalf("folders = %#v, want [Video]", folders)
	}
}

func TestDeleteFolder_MovesFeedsToOtherAndRemovesEmptyFolder(t *testing.T) {
	d := openTestDB(t)
	f, _ := d.UpsertFeed("A", "https://a.example/rss", "YouTube")
	if err := d.CreateFolder("YouTube"); err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}

	if err := d.DeleteFolder("YouTube"); err != nil {
		t.Fatalf("DeleteFolder: %v", err)
	}
	feeds, _ := d.ListFeeds()
	if got := feedByID(feeds, f.ID).Category; got != "" {
		t.Fatalf("category = %q, want empty", got)
	}
	folders, _ := d.ListFolders()
	if len(folders) != 0 {
		t.Fatalf("folders = %#v, want empty", folders)
	}
}

func feedByID(feeds []Feed, id int64) Feed {
	for _, f := range feeds {
		if f.ID == id {
			return f
		}
	}
	return Feed{}
}
