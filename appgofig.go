package appgofig

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type ConfigReadMode string

const (
	ReadModeEnvOnly     ConfigReadMode = "env-only"
	ReadModeYamlOnly    ConfigReadMode = "yaml-only"
	ReadModeEnvThenYaml ConfigReadMode = "env-yaml"
	ReadModeYamlThenEnv ConfigReadMode = "yaml-env"
)

type AppGofigOptions struct {
	ReadMode          ConfigReadMode
	YamlFilePath      string
	YamlFileRequested bool
	NewDefaults       map[string]string
}

type AppGofigOption func(*AppGofigOptions)

// WithReadMode sets a read mode
func WithReadMode(readMode ConfigReadMode) AppGofigOption {
	return func(options *AppGofigOptions) {
		options.ReadMode = readMode
	}
}

// WithYamlFile specifies which yaml file to use
func WithYamlFile(filePath string) AppGofigOption {
	return func(options *AppGofigOptions) {
		options.YamlFilePath = filePath
		options.YamlFileRequested = true
	}
}

// WithNewDefaults adds new default values to use
func WithNewDefaults(newDefaults map[string]string) AppGofigOption {
	return func(options *AppGofigOptions) {
		options.NewDefaults = newDefaults
	}
}

// ReadConfig takes your targetConfig struct, applies defaults and then applies values according to the readMode
// Using yamlFile, you can specify a yaml file to read from. If not specified, one of ./(config/)config.y(a)ml is used
func ReadConfig(targetConfig any, optionList ...AppGofigOption) error {
	if targetConfig == nil {
		return fmt.Errorf("targetConfig must not be nil")
	}

	if v := reflect.ValueOf(targetConfig); v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("targetConfig has to point to a struct")
	}

	// check if only the supported config types are present
	if err := onlyContainsSupportedTypes(targetConfig); err != nil {
		return fmt.Errorf("targetConfig not valid: %w", err)
	}

	// apply the options
	gofigOptions := &AppGofigOptions{
		ReadMode:          ReadModeEnvThenYaml,
		YamlFilePath:      "",
		YamlFileRequested: false,
		NewDefaults:       nil,
	}

	for _, opt := range optionList {
		opt(gofigOptions)
	}

	if gofigOptions.YamlFileRequested {
		if len(gofigOptions.YamlFilePath) == 0 {
			return fmt.Errorf("the yaml file path cannot be empty")
		}

		if gofigOptions.ReadMode == ReadModeEnvOnly {
			return fmt.Errorf("when using the ReadModeEnvOnly, no yaml file shall be specified")
		}
	}

	// apply the default values first
	if gofigOptions.NewDefaults == nil {
		if err := applyDefaultsToConfig(targetConfig); err != nil {
			return fmt.Errorf("unable to apply defaults: %w", err)
		}
	} else {
		if err := applyStringMapToConfig(targetConfig, gofigOptions.NewDefaults); err != nil {
			return fmt.Errorf("unable to apply new defaults: %w", err)
		}
	}

	// read the config according to the read mode
	switch gofigOptions.ReadMode {
	case ReadModeEnvOnly:
		// Only read from environment
		if err := applyEnvironmentToConfig(targetConfig); err != nil {
			return fmt.Errorf("could not read config values from env: %w", err)
		}
	case ReadModeYamlOnly:
		// Only read from yaml file
		if err := applyYamlToConfig(targetConfig, gofigOptions); err != nil {
			return fmt.Errorf("could not read config values from yaml: %w", err)
		}
	case ReadModeEnvThenYaml:
		// first read from environment, then overwrite existing stuff with yaml
		if err := applyEnvironmentToConfig(targetConfig); err != nil {
			return fmt.Errorf("could not read config values from env: %w", err)
		}
		if err := applyYamlToConfig(targetConfig, gofigOptions); err != nil {
			return fmt.Errorf("could not read config values from yaml: %w", err)
		}
	case ReadModeYamlThenEnv:
		// first read from yaml, then overwrite existing stuff from environment
		if err := applyYamlToConfig(targetConfig, gofigOptions); err != nil {
			return fmt.Errorf("could not read config values from yaml: %w", err)
		}
		if err := applyEnvironmentToConfig(targetConfig); err != nil {
			return fmt.Errorf("could not read config values from env: %w", err)
		}
	default:
		return fmt.Errorf("invalid read mode %s", gofigOptions.ReadMode)
	}

	// check if all required keys are non-empty
	if err := checkForEmptyRequiredFields(targetConfig); err != nil {
		return fmt.Errorf("missing required fields: %w", err)
	}

	return nil
}

// LogToConsole logs the actual configuration to the console
func LogConfig(targetConfig any, out io.Writer) {
	fmt.Fprint(out, "### AppGofig Configuration Start ###\n")

	t := reflect.TypeOf(targetConfig).Elem()
	v := reflect.ValueOf(targetConfig).Elem()

	for k := 0; k < t.NumField(); k++ {
		field := t.Field(k)
		val := v.Field(k)

		key := field.Name
		stringVal := readStringFromValue(val)

		if shouldBeMasked(key) {
			stringVal = fmt.Sprintf("[Masked - Length: %d]", len(stringVal))
		}

		fmt.Fprintf(out, "#| %s : %s\n", key, stringVal)
	}

	fmt.Fprint(out, "### AppGofig Configuration End ###\n")
}

// CreateMarkdownFile creates a simple markdown table with information about the provided config inputs
func WriteToMarkdownFile(targetConfig any, configDescriptions map[string]string, markdownFilePath string) error {
	var sb strings.Builder

	currentTimeString := time.Now().Format(time.RFC3339)

	sb.WriteString("# Default Configuration\n")
	fmt.Fprintf(&sb, "*Generated %s*\n\n", currentTimeString)

	sb.WriteString("| YAML Key | ENV Key | Type | Required | Default | Description |\n")
	sb.WriteString("|---|---|---|---|---|---|\n")

	t := reflect.TypeOf(targetConfig).Elem()
	for k := 0; k < t.NumField(); k++ {
		field := t.Field(k)
		yamlKey := field.Name
		envKey := field.Tag.Get("env")

		defaultValue := field.Tag.Get("default")
		description := configDescriptions[yamlKey]

		required := "no"
		if isRequiredField(field) {
			required = "yes"
		}

		// Write Markdown row
		sb.WriteString("| " + yamlKey + " | " + envKey + " | " + field.Type.Kind().String() + " | " + required + " | " + defaultValue + " | " + description + " |\n")
	}

	markdownFile, err := os.Create(markdownFilePath)
	if err != nil {
		return fmt.Errorf("unable to create config markdown file (%q): %w", markdownFilePath, err)
	}
	defer markdownFile.Close()

	if _, err := markdownFile.WriteString(sb.String()); err != nil {
		return fmt.Errorf("unable to write to config markdown file (%q): %w", markdownFilePath, err)
	}

	return nil
}

// CreateYamlExampleFile creates an example yaml file with comments providing the description and applied defaults
func WriteToYamlExampleFile(targetConfig any, configDescriptions map[string]string, yamlExampleFilePath string) error {
	var sb strings.Builder

	currentTimeString := time.Now().Format(time.RFC3339)

	sb.WriteString("# Autogenerated config.yml.example file. Please provide your own values here.\n")
	fmt.Fprintf(&sb, "# Generated %s \n\n", currentTimeString)

	t := reflect.TypeOf(targetConfig).Elem()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		yamlKey := field.Name

		defaultValue := field.Tag.Get("default")
		description := configDescriptions[field.Name]

		required := " - optional"
		if isRequiredField(field) {
			required = " - required"
		}

		// Write Row
		fmt.Fprintf(&sb, "# %s [%s%s] - %s \n", yamlKey, field.Type.Kind().String(), required, description)
		fmt.Fprintf(&sb, "%s: %s\n\n", yamlKey, defaultValue)
	}

	configExampleYaml, err := os.Create(yamlExampleFilePath)
	if err != nil {
		return fmt.Errorf("unable to create example config yaml file (%q): %w", yamlExampleFilePath, err)
	}
	defer configExampleYaml.Close()

	if _, err := configExampleYaml.WriteString(sb.String()); err != nil {
		return fmt.Errorf("unable to write example config yaml to file (%q): %w", yamlExampleFilePath, err)
	}

	return nil
}

// onlyContainsSupportedTypes checks if only supported data types are present within targtConfig
// if not, if returns an error describing the first non-valid field name
// This method assumes targetConfig to already be a pointer to struct
func onlyContainsSupportedTypes(targetConfig any) error {
	t := reflect.TypeOf(targetConfig).Elem()

	for k := 0; k < t.NumField(); k++ {
		field := t.Field(k)
		switch field.Type.Kind() {
		case reflect.String, reflect.Int64, reflect.Float64, reflect.Bool:
			continue
		default:
			return fmt.Errorf("invalid type %s on field %s", field.Type.Kind(), field.Name)
		}
	}

	return nil
}

// checkForEmptyRequiredFields returns an error if any field with req="true" tag has empty content
func checkForEmptyRequiredFields(targetConfig any) error {
	t := reflect.TypeOf(targetConfig).Elem()
	v := reflect.ValueOf(targetConfig).Elem()

	for k := 0; k < t.NumField(); k++ {
		field := t.Field(k)
		fieldVal := v.Field(k)
		switch field.Type.Kind() {
		case reflect.String:
			// only a string can be "empty" after the strconv methods were applied
			if isRequiredField(field) && len(fieldVal.String()) == 0 {
				return fmt.Errorf("required field %s has length 0", field.Name)
			}
		}
	}

	return nil
}

// shouldBeMasked returns true if key contains any of the key words
func shouldBeMasked(key string) bool {
	uppedKey := strings.ToUpper(key)

	if strings.HasSuffix(uppedKey, "PASSWORD") ||
		strings.HasSuffix(uppedKey, "TOKEN") ||
		strings.HasSuffix(uppedKey, "API_KEY") ||
		strings.HasSuffix(uppedKey, "SECRET") {
		return true
	}

	return false
}

// isRequiredField checks if field has a tag "req" and returns true only
// if that req is ok for strconv.ParseBool being true, false otherwise
func isRequiredField(field reflect.StructField) bool {
	reqVal, ok := field.Tag.Lookup("req")

	if !ok {
		return false
	}

	if boolVal, err := strconv.ParseBool(reqVal); err != nil {
		return false
	} else {
		return boolVal
	}
}
