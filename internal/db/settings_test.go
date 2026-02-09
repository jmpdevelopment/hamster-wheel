package db

import "testing"

func TestSetAndGetSetting(t *testing.T) {
	db := testDB(t)

	// Set a value.
	if err := db.SetSetting("poll_interval_minutes", "30"); err != nil {
		t.Fatalf("setting value: %v", err)
	}

	// Read it back.
	val, err := db.GetSetting("poll_interval_minutes")
	if err != nil {
		t.Fatalf("getting value: %v", err)
	}
	if val != "30" {
		t.Errorf("expected %q, got %q", "30", val)
	}
}

func TestGetSettingReturnsEmptyForMissing(t *testing.T) {
	db := testDB(t)

	val, err := db.GetSetting("nonexistent_key")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if val != "" {
		t.Errorf("expected empty string, got %q", val)
	}
}

func TestSetSettingOverwrites(t *testing.T) {
	db := testDB(t)

	db.SetSetting("theme", "light")
	db.SetSetting("theme", "dark")

	val, err := db.GetSetting("theme")
	if err != nil {
		t.Fatalf("getting value: %v", err)
	}
	if val != "dark" {
		t.Errorf("expected %q, got %q", "dark", val)
	}
}

func TestDeleteSetting(t *testing.T) {
	db := testDB(t)

	db.SetSetting("temp_key", "temp_value")

	if err := db.DeleteSetting("temp_key"); err != nil {
		t.Fatalf("deleting setting: %v", err)
	}

	val, err := db.GetSetting("temp_key")
	if err != nil {
		t.Fatalf("getting deleted setting: %v", err)
	}
	if val != "" {
		t.Errorf("expected empty string after delete, got %q", val)
	}
}

func TestDeleteSettingNonexistent(t *testing.T) {
	db := testDB(t)

	// Should not error when deleting a key that doesn't exist.
	if err := db.DeleteSetting("never_existed"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}
