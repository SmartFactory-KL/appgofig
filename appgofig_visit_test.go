package appgofig_test

import (
	"testing"

	"github.com/smartfactory-kl/appgofig"
)

func TestVisitConfigEntriesVisitsAllEntries(t *testing.T) {
	type TestConfig struct {
		Name   string `default:"app"`
		Port   int    `default:"8080"`
		Debug  bool   `default:"false"`
		Secret string `default:"secret"`
	}

	cfg := &TestConfig{
		Name:   "test-app",
		Port:   1234,
		Debug:  true,
		Secret: "top-secret",
	}

	var entries []appgofig.AppConfigEntry
	err := appgofig.VisitConfigEntries(cfg, func(entry appgofig.AppConfigEntry) {
		entries = append(entries, entry)
	})
	if err != nil {
		t.Fatalf("VisitConfigEntries() returned unexpected error: %v", err)
	}

	if len(entries) != 4 {
		t.Fatalf("expected VisitConfigEntries to create %d entries but got %d entries instead", 4, len(entries))
	}

	for _, entry := range entries {
		switch entry.Key {
		case "Name":
			if entry.Value != "test-app" {
				t.Fatalf("VisitConfigEntries changed a value from %v to %v", "test-app", entry.Value)
			}
		case "Port":
			if entry.Value != "1234" {
				t.Fatalf("VisitConfigEntries changed a value from %v to %v", 1234, entry.Value)
			}
		case "Debug":
			if entry.Value != "true" {
				t.Fatalf("VisitConfigEntries changed a value from %v to %v", true, entry.Value)
			}
		case "Secret":
			if entry.Value != "top-secret" {
				t.Fatalf("VisitConfigEntries changed a value from %v to %v", "top-secret", entry.Value)
			}
		default:
			t.Fatalf("VisitConfigEntries visited unexpected key %s", entry.Key)
		}
	}
}

func TestVisitConfigEntriesMasksValues(t *testing.T) {
	type TestConfig struct {
		Name       string  `default:"app"`
		Port       int     `default:"8080"`
		Debug      bool    `default:"false"`
		Secret     string  `default:"secret" masked:"true"`
		FloatValue float64 `default:"1"`
	}

	cfg := &TestConfig{
		Name:   "test-app",
		Port:   1234,
		Debug:  true,
		Secret: "top-secret",
	}

	var entries []appgofig.AppConfigEntry

	err := appgofig.VisitConfigEntries(cfg, func(entry appgofig.AppConfigEntry) {
		entries = append(entries, entry)
	})
	if err != nil {
		t.Fatalf("VisitConfigEntries() returned unexpected error: %v", err)
	}

	var secretEntry appgofig.AppConfigEntry
	for _, entry := range entries {
		if entry.Key == "Secret" {
			secretEntry = entry
			break
		}
	}

	if secretEntry.Value != "[Masked (len: 10)]" {
		t.Errorf("masked value = %q, want %q",
			secretEntry.Value,
			"[Masked (len: 10)]",
		)
	}

	if secretEntry.DefaultValue != "[Masked (len: 6)]" {
		t.Errorf("masked value = %q, want %q",
			secretEntry.DefaultValue, "[Masked (len: 6)]")
	}
}

func TestVisitConfigEntriesRejectsInvalidVisitFunc(t *testing.T) {
	type TestConfig struct {
		Name       string  `default:"app" env:"APP_NAME"`
		Port       int     `default:"8080" env:"APP_PORT"`
		Debug      bool    `default:"false" env:"APP_DEBUG"`
		Secret     string  `default:"secret" env:"APP_SECRET" secret:"true"`
		FloatValue float64 `default:"1"`
	}
	cfg := TestConfig{}

	err := appgofig.VisitConfigEntries(&cfg, nil)

	if err == nil {
		t.Fatalf("Expected error when using appgofig.VisitConfigEntries with invalid visit func")
	}
}
