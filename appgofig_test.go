package appgofig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type appgofigTestConfig struct {
	Name   string `default:"default-app" env:"APP_NAME"`
	Port   int    `default:"8080" required:"true"`
	Debug  bool   `default:"false"`
	Secret string `default:"secret" masked:"true"`

	BoolValue   bool    `default:""`
	IntValue    int     `default:"1"`
	FloatValue  float64 `default:"1.5"`
	StringValue string  `default:"HelloWorld"`
}

func TestReadConfigUsesDefaults(t *testing.T) {
	cfg := &appgofigTestConfig{}

	if err := ReadConfig(cfg); err != nil {
		t.Fatalf("ReadConfig() returned unexpected error: %v", err)
	}

	if cfg.Name != "default-app" {
		t.Errorf("AppName = %q, want %q", cfg.Name, "default-app")
	}

	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want %d", cfg.Port, 8080)
	}

	if cfg.Debug {
		t.Errorf("Debug = true, want false")
	}
}

func TestReadConfigRejectsUnexportedFields(t *testing.T) {
	type configWithPrivateField struct {
		Name   string `default:"app"`
		secret string `default:"hidden"`
	}

	err := ReadConfig(&configWithPrivateField{})

	if err == nil {
		t.Fatal("ReadConfig expected an error for an unexported field")
	}

	if !strings.Contains(err.Error(), "must be exported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVisitRejectsUnexportedFields(t *testing.T) {
	type configWithPrivateField struct {
		Name   string `default:"app"`
		secret string `default:"hidden"`
	}

	err := VisitConfigEntries(&configWithPrivateField{}, func(ace AppConfigEntry) {})

	if err == nil {
		t.Fatal("VisitConfigEntries expected an error for an unexported field")
	}
}

func TestReadConfigRejectsMalformedBool(t *testing.T) {
	t.Setenv("BOOL_VALUE", "nonsense")

	cfg := &appgofigTestConfig{}

	if err := ReadConfig(cfg, WithSources(EnvironmentSource())); err == nil {
		t.Fatalf("ReadConfig expected Error for malformed bool")
	}
}

func TestReadConfigRejectsMalformedInt(t *testing.T) {
	t.Setenv("INT_VALUE", "nonsense")

	cfg := &appgofigTestConfig{}

	if err := ReadConfig(cfg, WithSources(EnvironmentSource())); err == nil {
		t.Fatalf("ReadConfig expected Error for malformed int")
	}
}

func TestReadConfigRejectsMalformedFloat64(t *testing.T) {
	t.Setenv("FLOAT_VALUE", "nonsense")

	cfg := &appgofigTestConfig{}

	if err := ReadConfig(cfg, WithSources(EnvironmentSource())); err == nil {
		t.Fatalf("ReadConfig expected Error for malformed float64")
	}
}

func TestReadConfigAppliesEnvironmentValues(t *testing.T) {
	t.Setenv("APP_APP_NAME", "environment-app")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("APP_DEBUG", "true")

	cfg := &appgofigTestConfig{}

	err := ReadConfig(
		cfg,
		WithSources(
			PrefixedEnvironmentSource("APP"),
		),
	)
	if err != nil {
		t.Fatalf("ReadConfig() returned unexpected error: %v", err)
	}

	if cfg.Name != "environment-app" {
		t.Errorf("AppName = %q, want %q", cfg.Name, "environment-app")
	}

	if cfg.Port != 9090 {
		t.Errorf("Port = %d, want %d", cfg.Port, 9090)
	}

	if !cfg.Debug {
		t.Errorf("Debug = false, want true")
	}

	if cfg.Secret != "secret" {
		t.Errorf("Secret = %q, want %q", cfg.Secret, "secret")
	}
}

func TestReadConfigAppliesYAMLValues(t *testing.T) {
	t.Chdir(t.TempDir())

	yamlPath := filepath.Join("config.yaml")
	yamlContent := []byte("Secret: yaml-secret\nFloatValue: 2.5\n")

	if err := os.WriteFile(yamlPath, yamlContent, 0600); err != nil {
		t.Fatalf("failed to write test YAML: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Remove(yamlPath); err != nil {
			t.Log("failed to remove test file")
		}
	})

	cfg := &appgofigTestConfig{}

	err := ReadConfig(
		cfg,
		WithSources(YAMLSource()),
	)
	if err != nil {
		t.Fatalf("ReadConfig() returned unexpected error: %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("Port = %q, want default value of %q", cfg.Port, 8080)
	}

	if cfg.Secret != "yaml-secret" {
		t.Errorf("Secret = %q, want %q", cfg.Secret, "secret")
	}

	if cfg.FloatValue != 2.5 {
		t.Errorf("FloatValue = %f, want %f", cfg.FloatValue, 2.5)
	}
}

func TestReadConfigAppliesEnvironmentAndYAMLValuesFromSpecificFile(t *testing.T) {
	t.Setenv("APP_APP_NAME", "environment-app")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("APP_DEBUG", "true")
	t.Setenv("APP_SECRET", "env-secret")

	yamlPath := filepath.Join(t.TempDir(), "my_own_config.yml")
	yamlContent := []byte("Secret: yaml-secret\n")

	if err := os.WriteFile(yamlPath, yamlContent, 0600); err != nil {
		t.Fatalf("failed to write test YAML: %v", err)
	}

	cfg := &appgofigTestConfig{}

	err := ReadConfig(
		cfg,
		WithSources(
			PrefixedEnvironmentSource("APP"),
			SpecificYAMLSource(yamlPath),
		),
	)
	if err != nil {
		t.Fatalf("ReadConfig() returned unexpected error: %v", err)
	}

	if cfg.Name != "environment-app" {
		t.Errorf("AppName = %q, want %q", cfg.Name, "environment-app")
	}

	if cfg.Port != 9090 {
		t.Errorf("Port = %d, want %d", cfg.Port, 9090)
	}

	if !cfg.Debug {
		t.Errorf("Debug = false, want true")
	}

	if cfg.Secret != "yaml-secret" {
		t.Errorf("Secret = %q, want %q", cfg.Secret, "secret")
	}
}

func TestReadConfigAppliesOverridesLast(t *testing.T) {
	t.Setenv("APP_PORT", "9090")

	cfg := &appgofigTestConfig{}

	err := ReadConfig(
		cfg,
		WithSources(
			PrefixedEnvironmentSource("APP"),
		),
		WithOverrides(map[string]string{
			"Port": "7070",
		}),
	)
	if err != nil {
		t.Fatalf("ReadConfig() returned unexpected error: %v", err)
	}

	if cfg.Port != 7070 {
		t.Errorf("Port = %d, want %d", cfg.Port, 7070)
	}
}

func TestReadConfigRejectsMissingOverrideKey(t *testing.T) {
	cfg := &appgofigTestConfig{}

	err := ReadConfig(
		cfg,
		WithOverrides(map[string]string{
			"NotExisting": "7070",
		}),
	)
	if err == nil {
		t.Fatalf("ReadConfig() expected error for non-existing override key")
	}
}

func TestReadConfigRejectsEmptyRequiredOverrideKey(t *testing.T) {
	type MiniConfig struct {
		Port int `default:"8080" required:"true"`
	}

	err := ReadConfig(
		&MiniConfig{},
		WithOverrides(map[string]string{
			"Port": "",
		}),
	)
	if err == nil {
		t.Fatalf("ReadConfig() expected error for empty required override key")
	}
}

func TestReadConfigChecksRequiredFields(t *testing.T) {
	type appgofigTestWithRequired struct {
		AppName       string `default:"default-app" env:"APP_NAME"`
		RequiredInput string `required:"true"`
	}

	err := ReadConfig(&appgofigTestWithRequired{})

	if err == nil {
		t.Fatal("ReadCOnfig() expected an error for missing required fields")
	}

	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadConfigRejectsInvalidFieldType(t *testing.T) {
	type appgofigTestWithRequired struct {
		AppName       []int   `default:"default-app" env:"APP_NAME"`
		RequiredInput float32 `required:"true"`
	}

	err := ReadConfig(&appgofigTestWithRequired{})

	if err == nil {
		t.Fatal("ReadCOnfig() expected an error for invalid field type")
	}

	if !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type ErrorSource struct{}

func (src *ErrorSource) Load(map[string]*AppConfigEntry) (map[string]string, error) {
	return nil, fmt.Errorf("test error!")
}

func TestReadConfigRejectsOptionError(t *testing.T) {
	errSource := ErrorSource{}
	err := ReadConfig(&appgofigTestConfig{}, WithSources(&errSource))

	if err == nil {
		t.Fatal("ReadCOnfig() expected an error for an option erroring out")
	}
}

func TestVisitConfigEntriesMasksValues(t *testing.T) {
	cfg := &appgofigTestConfig{
		Name:   "test-app",
		Port:   1234,
		Debug:  true,
		Secret: "top-secret",
	}

	var entries []AppConfigEntry

	err := VisitConfigEntries(cfg, func(entry AppConfigEntry) {
		entries = append(entries, entry)
	})
	if err != nil {
		t.Fatalf("VisitConfigEntries() returned unexpected error: %v", err)
	}

	var secretEntry AppConfigEntry
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
}

func TestVisitConfigEntriesRejectsInvalidVisitFunc(t *testing.T) {
	cfg := appgofigTestConfig{}

	err := VisitConfigEntries(&cfg, nil)

	if err == nil {
		t.Fatalf("Expected error when using VisitConfigEntries with invalid visit func")
	}
}

func TestCreateConfigDocumentation(t *testing.T) {
	cfg := &appgofigTestConfig{}
	outputDir := t.TempDir()

	descriptions := map[string]string{
		"AppName": "The application name.",
		"Port":    "The application port.",
	}

	if err := CreateConfigDocumentation(cfg, descriptions, outputDir); err != nil {
		t.Fatalf("CreateConfigDocumentation() returned unexpected error: %v", err)
	}

	markdownPath := filepath.Join(outputDir, "DefaultDocumentation.md")
	yamlPath := filepath.Join(outputDir, "config.example.yaml")

	for _, path := range []string{markdownPath, yamlPath} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected documentation file %q: %v", path, err)
		}
	}
}

func TestCreateConfigDocumentationRejectsInvalidInput(t *testing.T) {
	cfg := appgofigTestConfig{}
	if err := CreateConfigDocumentation(&cfg, nil, ""); err == nil {
		t.Fatalf("CreateConfigDocumentation expected error in invalid config")
	}
}

func TestCreateConfigExampleYAMLRejectsInvalidInput(t *testing.T) {
	cfg := appgofigTestConfig{}
	if err := CreateConfigExampleYAML(&cfg, nil, ""); err == nil {
		t.Fatalf("CreateConfigDocumentation expected error in invalid config")
	}
}

func TestCreateConfigMarkdownDocumentRejectsInvalidInput(t *testing.T) {
	cfg := appgofigTestConfig{}
	if err := CreateConfigMarkdownDocument(&cfg, nil, ""); err == nil {
		t.Fatalf("CreateConfigDocumentation expected error in invalid config")
	}
}
