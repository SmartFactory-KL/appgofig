package appgofig

/**
 * Disclaimer: AI was used to generate parts of these tests.
 */

import (
	"os"
	"strings"
	"testing"
)

type TestConfig struct {
	StringVal string  `default:"defaultStr" env:"TEST_STRING" req:"true"`
	IntVal    int     `default:"42" env:"TEST_INT"`
	BoolVal   bool    `default:"true" env:"TEST_BOOL"`
	SecretVal string  `default:"topsecret" env:"TEST_SECRET" mask:"true"`
	FloatVal  float64 `default:"0.1" env:"TEST_FLOAT"`
}

type TestEmptyEnvTagConfig struct {
	StringVal string `default:"defaultStr" env:""`
}

// Helper to reset environment variables
func resetEnv() {
	os.Unsetenv("TEST_STRING")
	os.Unsetenv("TEST_INT")
	os.Unsetenv("TEST_BOOL")
	os.Unsetenv("TEST_SECRET")
	os.Unsetenv("TEST_FLOAT")
}

func TestInvalids(t *testing.T) {
	resetEnv()

	var cfg any
	err := ReadConfig(cfg)
	if err == nil {
		t.Fatal("expected any to not work, got no error instead")
	}

	newCfg := TestConfig{}
	err = ReadConfig(newCfg)
	if err == nil {
		t.Fatal("expected non-pointer to not work, got no error instead")
	}

	type TestInvalidType struct {
		StringVal    string `default:"defaultString"`
		NonValidType any    `default:"0"`
	}
	invalidCfg := &TestInvalidType{}
	err = ReadConfig(invalidCfg)
	if err == nil {
		t.Fatalf("expected invalid type to not work, git no error instead")
	}
}

func TestDefaults(t *testing.T) {
	resetEnv()
	cfg := &TestConfig{}
	err := ReadConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.StringVal != "defaultStr" {
		t.Errorf("expected StringVal=defaultStr, got %s", cfg.StringVal)
	}
	if cfg.IntVal != 42 {
		t.Errorf("expected IntVal=42, got %d", cfg.IntVal)
	}
	if cfg.BoolVal != true {
		t.Errorf("expected BoolVal=true, got %v", cfg.BoolVal)
	}
	if cfg.FloatVal != 0.1 {
		t.Errorf("expected FloatVal=true, got %v", cfg.FloatVal)
	}
}

func TestRequiredField(t *testing.T) {
	resetEnv()
	cfg := &TestConfig{}

	os.Setenv("TEST_STRING", "")

	err := ReadConfig(cfg, WithReadMode(ReadModeEnvOnly))
	if err == nil {
		t.Fatal("expected error due to required string, got nil")
	}
	if !strings.Contains(err.Error(), "required field StringVal") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestLogMasking(t *testing.T) {
	resetEnv()
	cfg := &TestConfig{}
	ReadConfig(cfg)

	var sb strings.Builder
	LogConfig(cfg, &sb)
	logOutput := sb.String()

	if !strings.Contains(logOutput, "[Masked") {
		t.Errorf("expected secret field to be masked, got: %s", logOutput)
	}
}

func TestInvalidReadModeYamlCombination(t *testing.T) {
	resetEnv()

	cfg := &TestConfig{}
	err := ReadConfig(cfg, WithReadMode(ReadModeEnvOnly), WithYamlFile("config.yml"))
	if err == nil {
		t.Fatal("expected error, got none")
	}
}

func TestYamlExampleAndMarkdownGeneration(t *testing.T) {
	resetEnv()
	cfg := &TestConfig{}
	ReadConfig(cfg)

	mdFile := "test.md"
	yamlFile := "test.yaml"

	defer os.Remove(mdFile)
	defer os.Remove(yamlFile)

	configDescriptions := map[string]string{
		"StringVal": "A required string value",
		"IntVal":    "An integer value",
		"BoolVal":   "A boolean value",
		"SecretVal": "Should be masked",
	}

	if err := WriteToMarkdownFile(cfg, configDescriptions, mdFile); err != nil {
		t.Fatalf("WriteToMarkdownFile failed: %v", err)
	}

	if err := WriteToYamlExampleFile(cfg, configDescriptions, yamlFile); err != nil {
		t.Fatalf("WriteToYamlExampleFile failed: %v", err)
	}

	if _, err := os.Stat(mdFile); err != nil {
		t.Fatalf("Markdown file not created: %v", err)
	}
	if _, err := os.Stat(yamlFile); err != nil {
		t.Fatalf("YAML example file not created: %v", err)
	}
}

func TestReadModeEnvOnly(t *testing.T) {
	resetEnv()
	os.Setenv("TEST_STRING", "envOnly")
	cfg := &TestConfig{}
	err := ReadConfig(cfg, WithReadMode(ReadModeEnvOnly))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.StringVal != "envOnly" {
		t.Errorf("expected StringVal=envOnly, got %s", cfg.StringVal)
	}
}

func TestEmptyEnvTagFallsBackToFieldName(t *testing.T) {
	resetEnv()
	os.Setenv("StringVal", "fieldNameEnv")

	cfg := &TestEmptyEnvTagConfig{}
	if err := ReadConfig(cfg, WithReadMode(ReadModeEnvOnly)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.StringVal != "fieldNameEnv" {
		t.Errorf("expected StringVal=fieldNameEnv, got %s", cfg.StringVal)
	}
}

func TestReadModeYamlOnly(t *testing.T) {
	resetEnv()

	cfg := &TestConfig{}

	yamlFile, err := os.Create("config.yml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(yamlFile.Name())

	yamlFile.WriteString("IntVal: 1000")

	if err := ReadConfig(cfg, WithReadMode(ReadModeYamlOnly)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.IntVal != 1000 {
		t.Errorf("expected IntVal=1000, got %d", cfg.IntVal)
	}
}

func TestReadModeYamlOnlyWithSpecifiedFile(t *testing.T) {
	resetEnv()

	cfg := &TestConfig{}

	yamlFile, err := os.Create("my_own_file.yml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(yamlFile.Name())

	yamlFile.WriteString("IntVal: 2000")

	if err := ReadConfig(cfg, WithReadMode(ReadModeYamlOnly), WithYamlFile("my_own_file.yml")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.IntVal != 2000 {
		t.Errorf("expected IntVal=2000, got %d", cfg.IntVal)
	}
}

func TestReadModeYamlOnlyWithInvalidFile(t *testing.T) {
	resetEnv()
	cfg := &TestConfig{}
	if err := ReadConfig(cfg, WithReadMode(ReadModeYamlOnly), WithYamlFile("")); err == nil {
		t.Fatal("expected error but got none")
	}

	if err := ReadConfig(cfg, WithReadMode(ReadModeYamlOnly), WithYamlFile("non-existing.yml")); err == nil {
		t.Fatal("expected error but got none")
	}

	yamlFile, err := os.Create("my_own_file.yml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(yamlFile.Name())

	invalidYaml := "# Missing colon after key\nStringVal \"missing_colon\"\n\n# Invalid indentation\nIntVal:\n  - 42\n   - 43\n\n# Unexpected character\nBoolVal: tru!!"
	yamlFile.WriteString(invalidYaml)

	if err := ReadConfig(cfg, WithReadMode(ReadModeYamlOnly), WithYamlFile("my_own_file.yml")); err == nil {
		t.Fatal("expected error but got none")
	}
}

func TestReadModeYamlThenEnv(t *testing.T) {
	resetEnv()
	cfg := &TestConfig{}

	// Env overrides
	os.Setenv("TEST_STRING", "envVal")
	os.Setenv("TEST_INT", "777")

	// yaml starts
	yamlFile, err := os.Create("config.yml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(yamlFile.Name())

	yamlFile.WriteString("IntVal: 1000\n")
	yamlFile.WriteString("StringVal: yamlVal\n")

	err = ReadConfig(cfg, WithReadMode(ReadModeYamlThenEnv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.StringVal != "envVal" {
		t.Errorf("expected StringVal=envVal, got %s", cfg.StringVal)
	}
	if cfg.IntVal != 777 {
		t.Errorf("expected IntVal=777, got %d", cfg.IntVal)
	}
}

func TestReadModeEnvThenYaml(t *testing.T) {
	resetEnv()
	cfg := &TestConfig{}

	// Env overrides
	os.Setenv("TEST_STRING", "envVal")
	os.Setenv("TEST_INT", "777")

	// yaml starts
	yamlFile, err := os.Create("config.yml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(yamlFile.Name())

	yamlFile.WriteString("IntVal: 1000\n")
	yamlFile.WriteString("StringVal: yamlVal\n")

	err = ReadConfig(cfg, WithReadMode(ReadModeEnvThenYaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.StringVal != "yamlVal" {
		t.Errorf("expected StringVal=yamlVal, got %s", cfg.StringVal)
	}
	if cfg.IntVal != 1000 {
		t.Errorf("expected IntVal=1000, got %d", cfg.IntVal)
	}
}

func TestMapInputOnly(t *testing.T) {
	resetEnv()
	cfg := &TestConfig{}

	mapInputVals := map[string]string{
		"StringVal": "custom",
		"IntVal":    "999",
	}
	err := ReadConfig(cfg,
		WithReadMode(ReadModeMapInputOnly),
		WithMapInput(mapInputVals),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.StringVal != "custom" {
		t.Errorf("expected StringVal=custom, got %s", cfg.StringVal)
	}
	if cfg.IntVal != 999 {
		t.Errorf("expected IntVal=999, got %d", cfg.IntVal)
	}
}

func TestMapInputIncorrectReadMode(t *testing.T) {
	resetEnv()
	cfg := &TestConfig{}

	mapInputVals := map[string]string{
		"StringVal": "custom",
		"IntVal":    "999",
	}
	err := ReadConfig(cfg,
		WithReadMode(ReadModeEnvOnly),
		WithMapInput(mapInputVals),
	)
	if err == nil {
		t.Fatalf("expected error, got none")
	}
}

func TestBooleanParsing(t *testing.T) {
	resetEnv()
	cfg := &TestConfig{}
	os.Setenv("TEST_BOOL", "true")
	err := ReadConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BoolVal != true {
		t.Errorf("expected BoolVal=true, got %v", cfg.BoolVal)
	}

	os.Setenv("TEST_BOOL", "false")
	err = ReadConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BoolVal != false {
		t.Errorf("expected BoolVal=false, got %v", cfg.BoolVal)
	}
}

func TestIntParsingErrors(t *testing.T) {
	resetEnv()
	os.Setenv("TEST_INT", "notanumber")
	cfg := &TestConfig{}
	err := ReadConfig(cfg)
	if err == nil {
		t.Fatal("expected error for invalid int value, got nil")
	}
	if !strings.Contains(err.Error(), "cannot use notanumber as int") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFloatParsingErrors(t *testing.T) {
	resetEnv()
	os.Setenv("TEST_FLOAT", "notanumber")
	cfg := &TestConfig{}
	err := ReadConfig(cfg)
	if err == nil {
		t.Fatal("expected error for invalid int value, got nil")
	}
	if !strings.Contains(err.Error(), "cannot use notanumber as float") {
		t.Errorf("unexpected error: %v", err)
	}
}
