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
	ReadModeMapInputOnly ConfigReadMode = "map-input-only" // only reading default values, intended for tests
	ReadModeEnvOnly      ConfigReadMode = "env-only"       // only reading environment
	ReadModeYamlOnly     ConfigReadMode = "yaml-only"      // only reading yaml
	ReadModeEnvThenYaml  ConfigReadMode = "env-yaml"       // first env, then yaml
	ReadModeYamlThenEnv  ConfigReadMode = "yaml-env"       // first yaml, then env
)

type AppGofigOptions struct {
	ReadMode          ConfigReadMode
	YamlFilePath      string
	YamlFileRequested bool
	MapInputValues    map[string]string
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

// WithMapInput adds new default values to use
func WithMapInput(values map[string]string) AppGofigOption {
	return func(options *AppGofigOptions) {
		options.MapInputValues = values
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
		MapInputValues:    nil,
	}

	for _, opt := range optionList {
		opt(gofigOptions)
	}

	// if a yaml file is requested, make sure it is
	// a) actually needed
	// b) not empty
	if gofigOptions.YamlFileRequested {
		if gofigOptions.ReadMode == ReadModeMapInputOnly {
			return fmt.Errorf("when using the ReadModeMapInputOnly, no yaml file shall be specified")
		}

		if gofigOptions.ReadMode == ReadModeEnvOnly {
			return fmt.Errorf("when using the ReadModeEnvOnly, no yaml file shall be specified")
		}

		if len(gofigOptions.YamlFilePath) == 0 {
			return fmt.Errorf("the yaml file path cannot be empty")
		}
	}

	// for the map input read mode, a mapInputValues cannot be nil
	if gofigOptions.ReadMode == ReadModeMapInputOnly {
		if gofigOptions.MapInputValues == nil {
			return fmt.Errorf("when using the ReadModeMapInputOnly, a non-nil map has to be provided via WithMapInput")
		}
	}

	// apply the default values first
	if err := applyDefaultsToConfig(targetConfig); err != nil {
		return fmt.Errorf("unable to apply default values: %w", err)
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
	case ReadModeMapInputOnly:
		// defaults have already been applied, now walk over map and apply values
		if err := applyStringMapToConfig(targetConfig, gofigOptions.MapInputValues); err != nil {
			return fmt.Errorf("unable to apply map input values: %w", err)
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
	if targetConfig == nil {
		fmt.Fprint(out, "### AppGofig Configuration Start ###\n")
		fmt.Fprint(out, "### ERROR: Config was nil")
		return
	}

	fmt.Fprint(out, "### AppGofig Configuration Start ###\n")

	t := reflect.TypeOf(targetConfig).Elem()
	v := reflect.ValueOf(targetConfig).Elem()

	for k := 0; k < t.NumField(); k++ {
		field := t.Field(k)
		val := v.Field(k)

		key := field.Name
		stringVal := readStringFromValue(val)

		if shouldBeMasked(field) {
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
		envKey := strings.TrimSpace(field.Tag.Get("env"))
		if len(envKey) == 0 {
			envKey = field.Name
		}

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
		case reflect.String, reflect.Int, reflect.Float64, reflect.Bool:
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

// shouldBeMasked returns true if the field has a "mask" tag
func shouldBeMasked(field reflect.StructField) bool {
	maskTag, hasMaskTag := field.Tag.Lookup("mask")
	if !hasMaskTag {
		return false
	}

	boolVal, _ := strconv.ParseBool(maskTag)

	return boolVal
}

// isRequiredField checks if field has a tag "req" or "required" and returns true only
// if one of them is ok for strconv.ParseBool being true, false otherwise.
// "req" takes prio if both are present.
func isRequiredField(field reflect.StructField) bool {
	reqVal, ok := field.Tag.Lookup("req")
	if !ok {
		reqVal, ok = field.Tag.Lookup("required")
		if !ok {
			return false
		}
	}

	if boolVal, err := strconv.ParseBool(reqVal); err != nil {
		return false
	} else {
		return boolVal
	}
}
