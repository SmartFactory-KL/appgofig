package appgofig

import (
	"fmt"
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

type ConfigEntry struct {
	Key   string
	Value string
}

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

	// prevent WithMapInput from working with anything else other than ReadModeMapInputOnly
	if gofigOptions.ReadMode != ReadModeMapInputOnly && gofigOptions.MapInputValues != nil {
		return fmt.Errorf("WithMapInput shall only be used in combination with ReadModeMapInputOnly")
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

// VisitConfigEntries is used to decouple logging of the current entries from any log implementation
// This replaces the old LogConfig method
func VisitConfigEntries(targetConfig any, visit func(ConfigEntry)) error {
	if targetConfig == nil {
		return fmt.Errorf("config was nil")
	}
	if visit == nil {
		return fmt.Errorf("visit func must not be nil")
	}

	value := reflect.ValueOf(targetConfig)
	if value.Kind() != reflect.Pointer || value.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("config has to point to a struct")
	}

	v := value.Elem()
	t := v.Type()

	for k := 0; k < t.NumField(); k++ {
		field := t.Field(k)
		value := v.Field(k)

		stringVal := readStringFromValue(value)
		isMasked := shouldBeMasked(field)

		if isMasked {
			stringVal = fmt.Sprintf("[Masked - Length: %d]", len(stringVal))
		}

		visit(ConfigEntry{
			Key:   field.Name,
			Value: stringVal,
		})
	}

	return nil
}

// CreateMarkdownFile creates a simple markdown table with information about the provided config inputs
func WriteToMarkdownFile(targetConfig any, configDescriptions map[string]string, markdownFilePath string) error {
	if targetConfig == nil {
		return fmt.Errorf("unable to create config markdown file (%q): config is nil", markdownFilePath)
	}

	var sb strings.Builder

	currentTimeString := time.Now().Format(time.RFC3339)

	sb.WriteString("# Default Configuration Information\n")
	fmt.Fprintf(&sb, "*Generated %s*\n\n", currentTimeString)

	WriteMarkdownOverviewTable(targetConfig, configDescriptions, &sb)

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

// WriteMarkdownOverviewTable will write a markdown table containing all availabel configuration information
// into the provided string builder
func WriteMarkdownOverviewTable(targetConfig any, configDescriptions map[string]string, sb *strings.Builder) {
	sb.WriteString("## Configuration Overview\n")
	sb.WriteString("This is an auto-generated overview of all configuration information\n")
	sb.WriteString("\n")
	sb.WriteString("| YAML Key | ENV Key | Type | Required | Default | Description |\n")
	sb.WriteString("|---|---|---|---|---|---|\n")

	t := reflect.TypeOf(targetConfig).Elem()
	for k := 0; k < t.NumField(); k++ {
		field := t.Field(k)
		fieldType := field.Type.Kind()
		yamlKey := field.Name
		envKey := strings.TrimSpace(field.Tag.Get("env"))
		if len(envKey) == 0 {
			envKey = field.Name
		}

		defaultValue := field.Tag.Get("default")
		description := configDescriptions[yamlKey]

		// mask default values
		if fieldType == reflect.String {
			defaultValue = fmt.Sprintf("`%s`", defaultValue)
		}

		required := "no"
		if isRequiredField(field) {
			required = "yes"
		}

		// Write Markdown row
		sb.WriteString("| " + yamlKey + " | `" + envKey + "` | " + fieldType.String() + " | " + required + " | " + defaultValue + " | " + description + " |\n")
	}
}

// CreateYamlExampleFile creates an example yaml file with comments providing the description and applied defaults
func WriteToYamlExampleFile(targetConfig any, configDescriptions map[string]string, yamlExampleFilePath string) error {
	if targetConfig == nil {
		return fmt.Errorf("unable to create example config yaml file (%q): config is nil", yamlExampleFilePath)
	}

	var sb strings.Builder

	currentTimeString := time.Now().Format(time.RFC3339)

	sb.WriteString("# Autogenerated config.yml.example file. Please provide your own values here.\n")
	fmt.Fprintf(&sb, "# Generated %s \n\n", currentTimeString)

	t := reflect.TypeOf(targetConfig).Elem()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldType := field.Type.Kind()
		yamlKey := field.Name

		defaultValue := field.Tag.Get("default")
		description := configDescriptions[field.Name]

		if fieldType == reflect.String {
			defaultValue = strconv.Quote(defaultValue)
		}

		required := " - optional"
		if isRequiredField(field) {
			required = " - required"
		}

		// Write Row
		fmt.Fprintf(&sb, "# %s [%s%s] - %s \n", yamlKey, fieldType.String(), required, description)
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

// hasBooleanTagSet returns true if any of the tagsToCheck contains a string value that strconv.ParseBool would parse to true.
// returns false otherwise. Priority is: first ok tag in tagsToCheck gets the win.
func hasBooleanTagSet(field reflect.StructField, tagsToCheck []string) bool {
	if len(tagsToCheck) == 0 {
		return false
	}

	for _, tagName := range tagsToCheck {
		tagValue, ok := field.Tag.Lookup(tagName)
		if ok {
			if boolVal, err := strconv.ParseBool(tagValue); err != nil {
				return false
			} else {
				return boolVal
			}
		}
	}

	return false
}

// shouldBeMasked returns true if the field has a "mask" tag
func shouldBeMasked(field reflect.StructField) bool {
	return hasBooleanTagSet(field, []string{"masked", "mask"})
}

// isRequiredField checks if field has a tag "req" or "required" and returns true only
// if one of them is ok for strconv.ParseBool being true, false otherwise.
// "req" takes prio if both are present.
func isRequiredField(field reflect.StructField) bool {
	return hasBooleanTagSet(field, []string{"required", "req"})
}
