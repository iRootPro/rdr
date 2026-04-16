package db

import "testing"

func TestSmartFolders_InsertListRoundTrip(t *testing.T) {
	d := openTestDB(t)
	a, err := d.InsertSmartFolder("Inbox", "unread")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if a.ID == 0 {
		t.Fatal("expected non-zero id")
	}
	if a.Position != 0 {
		t.Fatalf("first folder position: got %d, want 0", a.Position)
	}
	b, err := d.InsertSmartFolder("Starred", "starred")
	if err != nil {
		t.Fatalf("insert 2: %v", err)
	}
	if b.Position != 1 {
		t.Fatalf("second folder position: got %d, want 1", b.Position)
	}

	folders, err := d.ListSmartFolders()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(folders) != 2 {
		t.Fatalf("len: got %d, want 2", len(folders))
	}
	if folders[0].Name != "Inbox" || folders[0].Query != "unread" {
		t.Fatalf("folder[0]: %+v", folders[0])
	}
	if folders[1].Name != "Starred" || folders[1].Query != "starred" {
		t.Fatalf("folder[1]: %+v", folders[1])
	}
}

func TestSmartFolders_Update(t *testing.T) {
	d := openTestDB(t)
	f, _ := d.InsertSmartFolder("Inbox", "unread")
	if err := d.UpdateSmartFolder(f.ID, "Unread today", "unread today"); err != nil {
		t.Fatalf("update: %v", err)
	}
	folders, _ := d.ListSmartFolders()
	if folders[0].Name != "Unread today" || folders[0].Query != "unread today" {
		t.Fatalf("after update: %+v", folders[0])
	}
	if folders[0].Position != 0 {
		t.Fatalf("position must survive update, got %d", folders[0].Position)
	}
}

func TestSmartFolders_Delete(t *testing.T) {
	d := openTestDB(t)
	a, _ := d.InsertSmartFolder("A", "unread")
	b, _ := d.InsertSmartFolder("B", "starred")
	if err := d.DeleteSmartFolder(a.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	folders, _ := d.ListSmartFolders()
	if len(folders) != 1 || folders[0].ID != b.ID {
		t.Fatalf("after delete: %+v", folders)
	}
	// Deleting non-existent id is a no-op.
	if err := d.DeleteSmartFolder(9999); err != nil {
		t.Fatalf("delete missing: %v", err)
	}
}
