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

func TestGetReadRetentionDays_DefaultsTo90(t *testing.T) {
	d := openTestDB(t)
	got, err := d.GetReadRetentionDays()
	if err != nil {
		t.Fatalf("GetReadRetentionDays: %v", err)
	}
	if got != 90 {
		t.Fatalf("default: got %d, want 90", got)
	}
}

func TestGetReadRetentionDays_InvalidFallsBackToDefault(t *testing.T) {
	d := openTestDB(t)
	if err := d.SetSetting("read_retention_days", "not-a-number"); err != nil {
		t.Fatalf("SetSetting: %v", err)
	}
	got, _ := d.GetReadRetentionDays()
	if got != 90 {
		t.Fatalf("invalid value: got %d, want 90", got)
	}
}

func TestSetReadRetentionDays_RoundtripsKnownValues(t *testing.T) {
	d := openTestDB(t)
	for _, want := range []int{0, 1, 30, 90, 365, 1000} {
		if err := d.SetReadRetentionDays(want); err != nil {
			t.Fatalf("Set %d: %v", want, err)
		}
		got, err := d.GetReadRetentionDays()
		if err != nil {
			t.Fatalf("Get after set %d: %v", want, err)
		}
		if got != want {
			t.Fatalf("roundtrip %d: got %d", want, got)
		}
	}
}

func TestSetReadRetentionDays_ClampsNegative(t *testing.T) {
	d := openTestDB(t)
	if err := d.SetReadRetentionDays(-5); err != nil {
		t.Fatalf("Set -5: %v", err)
	}
	got, _ := d.GetReadRetentionDays()
	if got != 0 {
		t.Fatalf("negative should clamp to 0, got %d", got)
	}
}
