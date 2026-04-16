package i18n

import (
	"reflect"
	"testing"
)

// TestAllFieldsFilled walks both language tables recursively and fails if
// any string field is empty. This is the safety net for adding a new
// Strings field: the Go compiler enforces a zero value, and this test
// enforces a non-empty translation for every language.
func TestAllFieldsFilled(t *testing.T) {
	for _, tc := range []struct {
		name string
		val  Strings
	}{
		{"en", en},
		{"ru", ru},
	} {
		t.Run(tc.name, func(t *testing.T) {
			walk(t, tc.name, reflect.ValueOf(tc.val))
		})
	}
}

func walk(t *testing.T, path string, v reflect.Value) {
	switch v.Kind() {
	case reflect.Struct:
		rt := v.Type()
		for i := 0; i < v.NumField(); i++ {
			walk(t, path+"."+rt.Field(i).Name, v.Field(i))
		}
	case reflect.String:
		if v.String() == "" {
			t.Errorf("empty string at %s", path)
		}
	}
}

func TestForFallsBackToEN(t *testing.T) {
	if For("xx") != &en {
		t.Fatal("unknown lang should fall back to en")
	}
	if For(EN) != &en {
		t.Fatal("EN should return en table")
	}
	if For(RU) != &ru {
		t.Fatal("RU should return ru table")
	}
}

func TestParseDefaultsToEN(t *testing.T) {
	if Parse("") != EN {
		t.Fatal("empty string should parse to EN")
	}
	if Parse("en") != EN {
		t.Fatal("'en' should parse to EN")
	}
	if Parse("ru") != RU {
		t.Fatal("'ru' should parse to RU")
	}
	if Parse("garbage") != EN {
		t.Fatal("garbage should parse to EN")
	}
}
