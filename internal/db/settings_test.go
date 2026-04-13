package db

import "testing"

func TestSettings_ReadSeededDefaults(t *testing.T) {
	d := openTestDB(t)
	v, err := d.GetSetting("refresh_interval")
	if err != nil {
		t.Fatalf("GetSetting: %v", err)
	}
	if v != "30" {
		t.Fatalf("want 30, got %q", v)
	}
}

func TestSettings_SetOverwrites(t *testing.T) {
	d := openTestDB(t)
	if err := d.SetSetting("theme", "light"); err != nil {
		t.Fatalf("SetSetting: %v", err)
	}
	v, err := d.GetSetting("theme")
	if err != nil {
		t.Fatalf("GetSetting: %v", err)
	}
	if v != "light" {
		t.Fatalf("want light, got %q", v)
	}
}

func TestSettings_GetMissingReturnsEmpty(t *testing.T) {
	d := openTestDB(t)
	v, err := d.GetSetting("nope")
	if err != nil {
		t.Fatalf("GetSetting: %v", err)
	}
	if v != "" {
		t.Fatalf("want empty, got %q", v)
	}
}
