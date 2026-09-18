package appgofig_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smartfactory-kl/appgofig"
)

func TestReadConfigUsesDefaults(t *testing.T) {
	type TestConfig struct {
		IntValue    int     `default:"100"`
		BoolValue   bool    `default:"true"`
		FloatValue  float64 `default:"3.141"`
		StringValue string  `default:"Hello World"`
	}

	cfg, err := appgofig.ReadConfig(&TestConfig{})
	if err != nil {
		t.Fatalf("ReadConfig() returned unexpected error: %v", err)
	}

	if cfg.IntValue != 100 {
		t.Fatalf("ReadConfig() failed to apply default int value: got %d, want %d", cfg.IntValue, 100)
	}

	if cfg.BoolValue != true {
		t.Fatalf("ReadConfig() failed to apply default bool value: got %v, want %v", cfg.BoolValue, true)
	}

	if cfg.FloatValue != 3.141 {
		t.Fatalf("ReadConfig() failed to apply default float value: got %f, want %f", cfg.FloatValue, 3.141)
	}

	if cfg.StringValue != "Hello World" {
		t.Fatalf("ReadConfig() failed to apply default string value: got %s, want %s", cfg.StringValue, "Hello World")
	}
}

func TestReadConfigRejectsUnexportedFields(t *testing.T) {
	type TestConfigPrivateFields struct {
		Name   string `default:"app"`
		secret string `default:"hidden"`
	}

	_, err := appgofig.ReadConfig(&TestConfigPrivateFields{})

	if err == nil {
		t.Fatal("appgofig.ReadConfig expected an error for an unexported field")
	}

	if !strings.Contains(err.Error(), "must be exported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVisitRejectsUnexportedFields(t *testing.T) {
	type TestConfigPrivateFields struct {
		Name   string `default:"app"`
		secret string `default:"hidden"`
	}

	err := appgofig.VisitConfigEntries(&TestConfigPrivateFields{}, func(_ appgofig.AppConfigEntry) {})

	if err == nil {
		t.Fatal("appgofig.VisitConfigEntries expected an error for an unexported field")
	}
}

func TestReadConfigRejectsMalformedInput(t *testing.T) {
	type TestConfig struct {
		BoolValue  bool    `default:"true"`
		FloatValue float64 `default:"3.141"`
		IntValue   int     `default:"10"`
	}

	tests := []struct {
		name   string
		envKey string
		value  string
	}{
		{name: "bool", envKey: "BOOL_VALUE", value: "nonsense"},
		{name: "float", envKey: "FLOAT_VALUE", value: "nonsense"},
		{name: "int", envKey: "INT_VALUE", value: "nonsense"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ensure only the field under test is malformed.
			t.Setenv("BOOL_VALUE", "")
			t.Setenv("FLOAT_VALUE", "")
			t.Setenv("INT_VALUE", "")
			t.Setenv(tt.envKey, tt.value)

			_, err := appgofig.ReadConfig(
				&TestConfig{},
				appgofig.WithSources(appgofig.EnvironmentSource()),
			)
			if err == nil {
				t.Fatalf("ReadConfig() expected an error for malformed %s", tt.envKey)
			}
		})
	}
}

func TestReadConfigRejectsInvalidFieldType(t *testing.T) {
	type appgofigTestWithRequired struct {
		AppName       []int   `default:"default-app" env:"APP_NAME"`
		RequiredInput float32 `required:"true"`
	}

	_, err := appgofig.ReadConfig(&appgofigTestWithRequired{})

	if err == nil {
		t.Fatal("ReadCOnfig() expected an error for invalid field type")
	}

	if !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadConfigZerosEmptyNumerics(t *testing.T) {
	type TestConfig struct {
		IntValue    int     `default:"1"`
		BoolValue   bool    `default:"1"`
		FloatValue  float64 `default:"1"`
		StringValue string  `default:""`
	}

	t.Setenv("INT_VALUE", "")
	t.Setenv("BOOL_VALUE", "")
	t.Setenv("FLOAT_VALUE", "")

	cfg, err := appgofig.ReadConfig(&TestConfig{}, appgofig.WithSources(appgofig.EnvironmentSource()))
	if err != nil {
		t.Fatalf("unexpected error for appgofig.ReadConfig on zero input: %v", err)
	}

	if cfg.IntValue != 0 || cfg.FloatValue != 0 || cfg.BoolValue != false {
		t.Fatalf("expected zero on all numerical values, but got %v %v %v", cfg.IntValue, cfg.FloatValue, cfg.BoolValue)
	}
}

func TestReadConfigAppliesEnvironmentValues(t *testing.T) {
	type TestConfig struct {
		Name       string  `default:"app"`
		Port       int     `default:"8080"`
		Debug      bool    `default:"false"`
		Secret     string  `default:"secret" secret:"true"`
		FloatValue float64 `default:"1"`
	}

	t.Setenv("NAME", "environment-app")
	t.Setenv("PORT", "9090")
	t.Setenv("DEBUG", "true")

	cfg, err := appgofig.ReadConfig(
		&TestConfig{},
		appgofig.WithSources(
			appgofig.EnvironmentSource(),
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
	type TestConfig struct {
		Name       string  `default:"app"`
		Port       int     `default:"8080"`
		Debug      bool    `default:"false"`
		Secret     string  `default:"secret" secret:"true"`
		FloatValue float64 `default:"1"`
	}

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

	cfg, err := appgofig.ReadConfig(
		&TestConfig{},
		appgofig.WithSources(appgofig.YAMLSource()),
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

func TestReadConfigAppliesYAMLValuesFromSpecificFile(t *testing.T) {
	type TestConfig struct {
		Name       string  `default:"app"`
		Port       int     `default:"8080"`
		Debug      bool    `default:"false"`
		Secret     string  `default:"secret" secret:"true"`
		FloatValue float64 `default:"1"`
	}

	yamlPath := filepath.Join(t.TempDir(), "my_own_config.yml")
	yamlContent := []byte("Secret: yaml-secret\nFloatValue: 3.141\n")

	if err := os.WriteFile(yamlPath, yamlContent, 0600); err != nil {
		t.Fatalf("failed to write test YAML: %v", err)
	}

	cfg, err := appgofig.ReadConfig(
		&TestConfig{},
		appgofig.WithSources(
			appgofig.SpecificYAMLSource(yamlPath),
		),
	)
	if err != nil {
		t.Fatalf("ReadConfig() returned unexpected error: %v", err)
	}

	if cfg.Secret != "yaml-secret" {
		t.Errorf("Secret = %q, want %q", cfg.Secret, "secret")
	}

	if cfg.FloatValue != 3.141 {
		t.Errorf("FloatValue = %v, want %v", cfg.FloatValue, 3.141)
	}
}

func TestReadConfigRejectsInvalidYAML(t *testing.T) {
	t.Chdir(t.TempDir())
	type TestConfig struct {
		Name string `default:"default-name"`
		Age  int    `default:"100"`
	}

	yamlPath := filepath.Join("config.yaml")

	invalidYAML := map[string]string{
		"unterminated_quote": `key: "`,
		"bad_escape":         `key: "\q"`,
		"duplicate_key":      "key: one\nkey: two",
		"unknown_anchor":     `key: *missing`,
		"unterminated_flow":  `{`,
		"bad_indentation":    "key:\n  - one\n   - two",
	}

	for invalidityType, yamlContent := range invalidYAML {
		if err := os.WriteFile(yamlPath, []byte(yamlContent), 0600); err != nil {
			t.Fatalf("failed to write test YAML: %v", err)
		}

		_, err := appgofig.ReadConfig(
			&TestConfig{},
			appgofig.WithSources(
				appgofig.SpecificYAMLSource(yamlPath),
			),
		)
		if err == nil {
			t.Fatalf("expected ReadConfig() error on invalid yaml input for type %s", invalidityType)
		}
	}
}

func TestReadConfigRejectsInvalidYAMLTypes(t *testing.T) {
	t.Chdir(t.TempDir())
	type TestConfig struct {
		Name string `default:"default-name"`
		Age  int    `default:"100"`
	}

	yamlPath := filepath.Join("config.yaml")

	invalidYAMLTypes := map[string]string{
		"nested_map":     "parent:\n  child: value",
		"sequence":       "items:\n  - one",
		"mixed":          "name: app\nports:\n  - 8080",
		"empty_sequence": "items: []",
		"sequence_alt":   "items: [one, two]",
		"empty_mapping":  "item: {}",
		"mapping":        "item: {name: app}",
		"nested_mapping": "item:\n  name: app",
		"aliased_list":   "source: &list [one]\nitems: *list",
	}

	for invalidityType, yamlContent := range invalidYAMLTypes {
		if err := os.WriteFile(yamlPath, []byte(yamlContent), 0600); err != nil {
			t.Fatalf("failed to write test YAML: %v", err)
		}

		_, err := appgofig.ReadConfig(
			&TestConfig{},
			appgofig.WithSources(
				appgofig.SpecificYAMLSource(yamlPath),
			),
		)
		if err == nil {
			t.Fatalf("expected ReadConfig() error on invalid yaml type input for type %s", invalidityType)
		}
	}
}

func TestReadConfigAppliesOverrides(t *testing.T) {
	type TestConfig struct {
		Name       string  `default:"app" env:"APP_NAME"`
		Port       int     `default:"8080" env:"APP_PORT"`
		Debug      bool    `default:"false" env:"APP_DEBUG"`
		Secret     string  `default:"secret" env:"APP_SECRET" secret:"true"`
		FloatValue float64 `default:"1"`
	}

	cfg, err := appgofig.ReadConfig(
		&TestConfig{},
		appgofig.WithOverrides(map[string]string{
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
	type TestConfig struct {
		Name       string  `default:"app" env:"APP_NAME"`
		Port       int     `default:"8080" env:"APP_PORT"`
		Debug      bool    `default:"false" env:"APP_DEBUG"`
		Secret     string  `default:"secret" env:"APP_SECRET" secret:"true"`
		FloatValue float64 `default:"1"`
	}
	_, err := appgofig.ReadConfig(
		&TestConfig{},
		appgofig.WithOverrides(map[string]string{
			"NotExisting": "7070",
		}),
	)
	if err == nil {
		t.Fatalf("ReadConfig() expected error for non-existing override key")
	}
}

func TestReadConfigChecksRequiredFields(t *testing.T) {
	type appgofigTestWithRequired struct {
		AppName       string `default:"default-app" env:"APP_NAME"`
		RequiredInput string `default:"" required:"true"`
	}

	_, err := appgofig.ReadConfig(&appgofigTestWithRequired{})

	if err == nil {
		t.Fatal("ReadCOnfig() expected an error for missing required fields")
	}

	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadConfigAppliesSourcesInOrder(t *testing.T) {
	t.Chdir(t.TempDir())
	type TestConfig struct {
		Name string `default:"default-name"`
		Age  int    `default:"100"`
	}

	t.Setenv("NAME", "env-name")
	t.Setenv("AGE", "200")

	yamlPath := filepath.Join("config.yaml")
	yamlContent := []byte("Name: yaml-name\nAge: 300\n")

	if err := os.WriteFile(yamlPath, yamlContent, 0600); err != nil {
		t.Fatalf("failed to write test YAML: %v", err)
	}

	firstOrderCfg, err := appgofig.ReadConfig(
		&TestConfig{},
		appgofig.WithSources(
			appgofig.EnvironmentSource(),
			appgofig.YAMLSource(),
		),
	)
	if err != nil {
		t.Fatalf("ReadConfig() returned unexpected error: %v", err)
	}

	if firstOrderCfg.Name != "yaml-name" {
		t.Fatalf("ENV-YAML-Order resultet in %q instead of %q for %q", firstOrderCfg.Name, "yaml-name", "Name")
	}

	if firstOrderCfg.Age != 300 {
		t.Fatalf("ENV-YAML-Order resultet in %q instead of %q for %q", firstOrderCfg.Age, 300, "Age")
	}

	secondOrderCfg, err := appgofig.ReadConfig(
		&TestConfig{},
		appgofig.WithSources(
			appgofig.YAMLSource(),
			appgofig.EnvironmentSource(),
		),
	)

	if err != nil {
		t.Fatalf("ReadConfig() returned unexpected error: %v", err)
	}

	if secondOrderCfg.Name != "env-name" {
		t.Fatalf("YAML-ENV-Order resultet in %q instead of %q for %q", secondOrderCfg.Name, "env-name", "Name")
	}

	if secondOrderCfg.Age != 200 {
		t.Fatalf("YAML-ENV-Order resultet in %q instead of %q for %q", secondOrderCfg.Age, 200, "Age")
	}
}
