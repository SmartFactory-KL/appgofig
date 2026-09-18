package appgofig_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smartfactory-kl/appgofig"
	"go.yaml.in/yaml/v3"
)

func TestCreateConfigDocumentation(t *testing.T) {
	type TestConfig struct {
		Name       string  `default:"app" env:"APP_NAME"`
		Port       int     `default:"8080" env:"APP_PORT"`
		Debug      bool    `default:"false" env:"APP_DEBUG"`
		Secret     string  `default:"secret" env:"APP_SECRET" secret:"true"`
		FloatValue float64 `default:"1"`
	}
	cfg := &TestConfig{}
	outputDir := t.TempDir()

	descriptions := map[string]string{
		"AppName": "The application name.",
		"Port":    "The application port.",
	}

	if err := appgofig.CreateConfigDocumentation(cfg, descriptions, "APP", outputDir); err != nil {
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
	type TestConfig struct {
		Name       string  `default:"app" env:"APP_NAME"`
		Port       int     `default:"8080" env:"APP_PORT"`
		Debug      bool    `default:"false" env:"APP_DEBUG"`
		Secret     string  `default:"secret" env:"APP_SECRET" secret:"true"`
		FloatValue float64 `default:"1"`
	}
	cfg := TestConfig{}
	if err := appgofig.CreateConfigDocumentation(&cfg, nil, "APP", ""); err == nil {
		t.Fatalf("appgofig.CreateConfigDocumentation expected error in invalid config")
	}
}

func TestCreateConfigExampleYAMLRejectsInvalidInput(t *testing.T) {
	type TestConfig struct {
		Name       string  `default:"app" env:"APP_NAME"`
		Port       int     `default:"8080" env:"APP_PORT"`
		Debug      bool    `default:"false" env:"APP_DEBUG"`
		Secret     string  `default:"secret" env:"APP_SECRET" secret:"true"`
		FloatValue float64 `default:"1"`
	}
	cfg := TestConfig{}
	if err := appgofig.CreateConfigExampleYAML(&cfg, nil, "APP", ""); err == nil {
		t.Fatalf("appgofig.CreateConfigDocumentation expected error in invalid config")
	}
}

func TestCreateConfigMarkdownDocumentRejectsInvalidInput(t *testing.T) {
	type TestConfig struct {
		Name       string  `default:"app" env:"APP_NAME"`
		Port       int     `default:"8080" env:"APP_PORT"`
		Debug      bool    `default:"false" env:"APP_DEBUG"`
		Secret     string  `default:"secret" env:"APP_SECRET" secret:"true"`
		FloatValue float64 `default:"1"`
	}
	cfg := TestConfig{}
	if err := appgofig.CreateConfigMarkdownDocument(&cfg, nil, "APP", ""); err == nil {
		t.Fatalf("appgofig.CreateConfigDocumentation expected error in invalid config")
	}
}

func TestCreateConfigDocumentationWritesExpectedContent(t *testing.T) {
	type TestConfig struct {
		Name       string `default:"demo" env:"NAME"`
		Port       int    `default:"8080" required:"true"`
		APIKey     string `default:"secret-value" env:"API_KEY" masked:"true"`
		RetryLimit int    `default:"42424242" env:"RETRY_LIMIT" masked:"true"`
		Disabled   bool   `default:"false"`
	}

	outputDir := t.TempDir()
	err := appgofig.CreateConfigDocumentation(
		&TestConfig{},
		map[string]string{
			"Name":       "The application name.",
			"Port":       "The listening port.",
			"APIKey":     "Credential for the upstream API.",
			"RetryLimit": "Maximum retry attempts.",
		},
		"APP",
		outputDir,
	)
	if err != nil {
		t.Fatalf("CreateConfigDocumentation() error = %v", err)
	}

	markdown, err := os.ReadFile(filepath.Join(outputDir, "DefaultDocumentation.md"))
	if err != nil {
		t.Fatalf("failed to read generated Markdown: %v", err)
	}

	for _, want := range []string{
		"| `Name` | `APP_NAME` | `string` | demo | `No` | The application name. |",
		"| `Port` | `APP_PORT` | `int` | 8080 | `Yes` | The listening port. |",
		"| `APIKey` | `APP_API_KEY` | `string` | [Masked (len: 12)] | `No` | Credential for the upstream API. |",
		"| `RetryLimit` | `APP_RETRY_LIMIT` | `int` | [Masked (len: 8)] | `No` | Maximum retry attempts. |",
		`      APP_NAME: "demo"`,
		`      APP_PORT: "8080"`,
		`      APP_API_KEY: "[Masked (len: 12)]"`,
		`      APP_RETRY_LIMIT: "[Masked (len: 8)]"`,
		`  -e APP_NAME='demo' \`,
		`  -e APP_PORT='8080' \`,
		`  -e APP_API_KEY='[Masked (len: 12)]' \`,
		`  -e APP_RETRY_LIMIT='[Masked (len: 8)]' \`,
		`  -e APP_DISABLED='false' \`,
		"  your-image:latest",
	} {
		if !strings.Contains(string(markdown), want) {
			t.Errorf("generated Markdown does not contain %q\n%s", want, markdown)
		}
	}

	for _, secret := range []string{"secret-value", "42424242"} {
		if strings.Contains(string(markdown), secret) {
			t.Errorf("generated Markdown exposes masked default %q:\n%s", secret, markdown)
		}
	}

	yamlContent, err := os.ReadFile(filepath.Join(outputDir, "config.example.yaml"))
	if err != nil {
		t.Fatalf("failed to read generated YAML: %v", err)
	}

	for _, want := range []string{
		"# Name [string]\n# Environment: APP_NAME\n# The application name.\nName: \"demo\"",
		"# Port [int - required]\n# Environment: APP_PORT\n# The listening port.\nPort: 8080",
		"# APIKey [string]\n# Environment: APP_API_KEY\n# Credential for the upstream API.\nAPIKey: \"[Masked (len: 12)]\"",
		"# RetryLimit [int]\n# Environment: APP_RETRY_LIMIT\n# Maximum retry attempts.\nRetryLimit: \"[Masked (len: 8)]\"",
		"# Disabled [bool]\n# Environment: APP_DISABLED\nDisabled: false",
	} {
		if !strings.Contains(string(yamlContent), want) {
			t.Errorf("generated YAML does not contain %q\n%s", want, yamlContent)
		}
	}

	for _, secret := range []string{"secret-value", "RetryLimit: 42424242"} {
		if strings.Contains(string(yamlContent), secret) {
			t.Errorf("generated YAML exposes masked default %q:\n%s", secret, yamlContent)
		}
	}

	var yamlMap map[string]string
	if err := yaml.Unmarshal(yamlContent, &yamlMap); err != nil {
		t.Errorf("generated YAML failed to unmarshal: %v", err)
	}
}
