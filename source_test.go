package appgofig

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestYAMLSource_Load(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		yaml    string
		config  map[string]*AppConfigEntry
		want    map[string]string
		wantErr bool
	}{
		{
			name: "loads matching keys and ignores unknown keys",
			yaml: `
app_name: test-app
port: "8080"
unknown: ignored
`,
			config: map[string]*AppConfigEntry{
				"app_name": {},
				"port":     {},
			},
			want: map[string]string{
				"app_name": "test-app",
				"port":     "8080",
			},
		},
		{
			name: "returns empty map when no keys match",
			yaml: `
unknown: value
`,
			config: map[string]*AppConfigEntry{
				"app_name": {},
			},
			want: map[string]string{},
		},
		{
			name: "handles empty yaml",
			yaml: "",
			config: map[string]*AppConfigEntry{
				"app_name": {},
			},
			want: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			filePath := filepath.Join(dir, "config.yaml")

			if err := os.WriteFile(
				filePath,
				[]byte(tt.yaml),
				0600,
			); err != nil {
				t.Fatalf("failed to write test YAML: %v", err)
			}

			source := YAMLSource(filePath)
			got, err := source.Load(tt.config)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Load() error = %v, wantErr = %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Load() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestYAMLSource_LoadMissingFile(t *testing.T) {
	t.Parallel()

	source := YAMLSource(filepath.Join(
		t.TempDir(),
		"missing.yaml",
	))

	_, err := source.Load(map[string]*AppConfigEntry{
		"app_name": {},
	})

	if err == nil {
		t.Fatal("Load() expected an error for a missing file")
	}
}

func TestEnvironmentSource_Load(t *testing.T) {
	t.Setenv("APP_APP_NAME", "test-app")
	t.Setenv("APP_PORT", "8080")
	t.Setenv("UNRELATED_KEY", "ignored")

	config := map[string]*AppConfigEntry{
		"app_name": {
			Key: "app_name",
		},
		"port": {
			Key: "port",
		},
	}

	source := EnvironmentSource("app")

	got, err := source.Load(config)
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	want := map[string]string{
		"app_name": "test-app",
		"port":     "8080",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Load() = %#v, want %#v", got, want)
	}
}

func TestEnvironmentSource_LoadExplicitEnvironmentKey(t *testing.T) {
	t.Setenv("APP_CUSTOM_APP_NAME", "custom-value")

	config := map[string]*AppConfigEntry{
		"app_name": {
			Key:            "app_name",
			EnvironmentKey: "CUSTOM_APP_NAME",
		},
	}

	source := EnvironmentSource("APP")

	got, err := source.Load(config)
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	want := map[string]string{
		"app_name": "custom-value",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Load() = %#v, want %#v", got, want)
	}
}

func TestEnvironmentSource_LoadIgnoresMissingVariables(t *testing.T) {
	t.Setenv("APP_EXISTING", "present")

	config := map[string]*AppConfigEntry{
		"existing": {
			Key: "existing",
		},
		"missing": {
			Key: "missing",
		},
	}

	source := EnvironmentSource("APP")

	got, err := source.Load(config)
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	want := map[string]string{
		"existing": "present",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Load() = %#v, want %#v", got, want)
	}
}

func TestEnvironmentSource_LoadDoesNotMutateConfig(t *testing.T) {
	t.Setenv("APP_PORT", "8080")

	config := map[string]*AppConfigEntry{
		"port": {
			EnvironmentKey: "",
		},
	}

	original := *config["port"]

	source := EnvironmentSource("APP")

	_, err := source.Load(config)
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if !reflect.DeepEqual(*config["port"], original) {
		t.Errorf("Load() mutated config entry: got %#v, want %#v",
			*config["port"],
			original,
		)
	}
}

func TestSourcesImplementAppGofigSource(t *testing.T) {
	t.Parallel()

	var _ AppGofigSource = YAMLSource()
	var _ AppGofigSource = EnvironmentSource("")
}
