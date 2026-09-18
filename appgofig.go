package appgofig

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

// ReadConfig constructs a configuration of type T from its tags, provided sources and overrides. Based on this it first reads the default values
// and then all sources in order, where later values overwrite earlier ones. It returns the Config with applied values.
func ReadConfig[T any](optionList ...AppGofigOption) (*T, error) {
	cfg := new(T)

	// apply the options
	gofigOptions := &AppGofigOptions{
		Sources:   nil,
		Overrides: nil,
	}
	for _, opt := range optionList {
		if opt != nil {
			opt(gofigOptions)
		}
	}

	// check validity of cfg input
	if err := checkConfigStruct(cfg); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// read defaults first
	entryMap := readConfigDefaults(cfg)

	// Load all sources in order
	sourceMaps := make([]map[string]string, 0, len(gofigOptions.Sources))
	if len(gofigOptions.Sources) != 0 {
		// validate sources
		for _, src := range gofigOptions.Sources {
			srcMap, err := src.Load(entryMap)
			if err != nil {
				return nil, err
			}

			sourceMaps = append(sourceMaps, srcMap)
		}
	}

	// Apply all source maps to resultConfig
	for _, sourceMap := range sourceMaps {
		for cfgKey := range entryMap {
			val, ok := sourceMap[cfgKey]
			if ok {
				entryMap[cfgKey].Value = val
			}
		}
	}

	// Apply overrides at the end
	if len(gofigOptions.Overrides) > 0 {
		for key, val := range gofigOptions.Overrides {
			targetEntry, ok := entryMap[key]
			if !ok {
				return nil, fmt.Errorf("Override contains key %s which does not exist on config", key)
			}

			targetEntry.Value = val
		}
	}

	// check if all required keys are non-empty
	if err := checkRequiredFields(entryMap); err != nil {
		return nil, fmt.Errorf("missing required fields: %w", err)
	}

	// apply values to a copy of config struct and return it

	v := reflect.ValueOf(cfg).Elem()
	t := v.Type()

	for k := 0; k < t.NumField(); k++ {
		field := t.Field(k)
		fieldVal := v.Field(k)

		curEntry, ok := entryMap[field.Name]
		if !ok {
			// Unsure whether this would ever happen
			// since valueMap should have defaults for every entry
			// of the config struct
			continue
		}

		if err := applyEntryToValue(field, fieldVal, curEntry); err != nil {
			return nil, fmt.Errorf("failed to write value %s to field %s : %w", curEntry.Value, field.Name, err)
		}
	}

	return cfg, nil
}

// VisitConfigEntries will run visit() once using AppConfigEntries with the actual value taken from cfg itself.
// Any values marked as "IsMasked" will be converted to "[Masked (len:x)]" with x being the string length
func VisitConfigEntries[T any](cfg *T, visit func(AppConfigEntry)) error {
	// check validity of cfg input
	if err := checkConfigStruct(cfg); err != nil {
		return fmt.Errorf("failed to visit config values: %w", err)
	}

	if visit == nil {
		return fmt.Errorf("visit function cannot be nil")
	}

	cfgValues := reflect.ValueOf(cfg).Elem()
	defaultValues := readConfigDefaults(cfg)

	// Preserve struct field order for deterministic output
	keys := getConfigEntryKeys(cfg)

	for _, cfgKey := range keys {
		entry := defaultValues[cfgKey]

		field := cfgValues.FieldByName(entry.Key)

		value := readStringFromValue(field)
		if entry.IsMasked {
			value = maskString(value)
		}

		defaultValue := getEntryMaskedDefaultValue(entry)

		visit(AppConfigEntry{
			Key:            entry.Key,
			Value:          value,
			DefaultValue:   defaultValue,
			ValueType:      entry.ValueType,
			IsRequired:     entry.IsRequired,
			IsMasked:       entry.IsMasked,
			EnvironmentKey: entry.EnvironmentKey,
		})
	}

	return nil
}

// CreateConfigDocumentation will create all config documents using default paths and put them into the outputDir, creating it if needed
func CreateConfigDocumentation[T any](cfg *T, cfgDescriptions map[string]string, envPrefix string, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory for config documentation: %w", err)
	}

	markdownPath := filepath.Join(outputDir, "DefaultDocumentation.md")
	if err := CreateConfigMarkdownDocument(cfg, cfgDescriptions, envPrefix, markdownPath); err != nil {
		return fmt.Errorf("failed to create config documentation: %w", err)
	}

	yamlPath := filepath.Join(outputDir, "config.example.yaml")
	if err := CreateConfigExampleYAML(cfg, cfgDescriptions, envPrefix, yamlPath); err != nil {
		return fmt.Errorf("failed to create config example YAML: %w", err)
	}

	return nil
}

// CreateConfigExampleYAML creates an example yaml file with all config entries, adding the metadata as comments
func CreateConfigExampleYAML[T any](cfg *T, cfgDescriptions map[string]string, envPrefix string, outputPath string) error {
	if err := checkConfigStruct(cfg); err != nil {
		return fmt.Errorf("failed to create documentation: %w", err)
	}

	if cfgDescriptions == nil {
		cfgDescriptions = make(map[string]string)
	}

	entryMap := readConfigDefaults(cfg)

	// Preserve struct field order for deterministic output
	keys := getConfigEntryKeys(cfg)

	var sb strings.Builder

	sb.WriteString("# Config Example YAML\n")
	sb.WriteString("# Auto generated file. Please provide your own values here\n\n")

	for _, cfgKey := range keys {
		cfgEntry := entryMap[cfgKey]

		cfgDescription := strings.TrimSpace(cfgDescriptions[cfgEntry.Key])
		isRequiredString := ""
		if cfgEntry.IsRequired {
			isRequiredString = " - required"
		}

		fmt.Fprintf(&sb, "# %s [%s%s]\n", cfgEntry.Key, cfgEntry.ValueType.String(), isRequiredString)
		fmt.Fprintf(&sb, "# Environment: %s\n", getEntryEnvKey(envPrefix, cfgEntry))

		if len(cfgDescription) > 0 {
			fmt.Fprintf(&sb, "# %s\n", cfgDescription)
		}

		defaultValue := getEntryMaskedDefaultValue(cfgEntry)

		// IsMasked will always result in a string value and therefore it should be quoted
		if cfgEntry.ValueType == reflect.String || cfgEntry.IsMasked {
			defaultValue = strconv.Quote(defaultValue)
		}

		fmt.Fprintf(&sb, "%s: %s\n\n", cfgEntry.Key, defaultValue)
	}

	sb.WriteString("# End of auto generated file\n")

	if err := os.WriteFile(outputPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("failed to create output file for yaml example: %w", err)
	}

	return nil
}

// CreateConfigMarkdownDocument creates a Markdown document containing:
// - A table with an overview of all config entries, including their environment key
// - An example environment block for a Docker Compose file
// - An example command for docker run containing all environment entries
func CreateConfigMarkdownDocument[T any](cfg *T, cfgDescriptions map[string]string, envPrefix string, outputPath string) error {
	if err := checkConfigStruct(cfg); err != nil {
		return fmt.Errorf("failed to create documentation: %w", err)
	}

	if cfgDescriptions == nil {
		cfgDescriptions = make(map[string]string)
	}

	entryMap := readConfigDefaults(cfg)

	// Preserve struct field order for deterministic output
	keys := getConfigEntryKeys(cfg)

	var sb strings.Builder

	sb.WriteString("# Configuration Documentation\n\n")
	sb.WriteString("> Auto-generated documentation. Do not edit manually.\n\n")

	// ---------------------------------------------------------
	// 1. Configuration overview
	// ---------------------------------------------------------

	sb.WriteString("## Configuration Overview\n\n")

	sb.WriteString("| Key | Environment Variable | Type | Default | Required | Description |\n")
	sb.WriteString("| --- | --- | --- | --- | --- | --- |\n")

	for _, key := range keys {
		entry := entryMap[key]
		description := strings.TrimSpace(cfgDescriptions[key])

		required := "No"
		if entry.IsRequired {
			required = "Yes"
		}

		defaultValue := getEntryMaskedDefaultValue(entry)

		fmt.Fprintf(
			&sb,
			"| `%s` | `%s` | `%s` | %s | `%s` | %s |\n",
			key,
			getEntryEnvKey(envPrefix, entry),
			entry.ValueType.String(),
			escapeMarkdown(defaultValue),
			required,
			escapeMarkdown(description),
		)
	}

	sb.WriteString("\n")

	// ---------------------------------------------------------
	// 2. Docker Compose environment block
	// ---------------------------------------------------------

	sb.WriteString("## Docker Compose Example\n\n")
	sb.WriteString("```yaml\n")
	sb.WriteString("services:\n")
	sb.WriteString("  app:\n")
	sb.WriteString("    environment:\n")

	for _, key := range keys {
		entry := entryMap[key]

		envKey := getEntryEnvKey(envPrefix, entry)
		if envKey == "" {
			continue
		}

		defaultValue := getEntryMaskedDefaultValue(entry)

		fmt.Fprintf(
			&sb,
			"      %s: %s\n",
			envKey,
			strconv.Quote(defaultValue),
		)
	}

	sb.WriteString("```\n\n")

	// ---------------------------------------------------------
	// 3. Docker run command
	// ---------------------------------------------------------

	sb.WriteString("## Docker Run Example\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("docker run -it \\\n")

	dockerEntries := make([]string, 0, len(keys))

	for _, key := range keys {
		entry := entryMap[key]

		envKey := getEntryEnvKey(envPrefix, entry)
		if envKey == "" {
			continue
		}

		defaultValue := getEntryMaskedDefaultValue(entry)

		dockerEntries = append(
			dockerEntries,
			fmt.Sprintf(
				"  -e %s=%s",
				envKey,
				shellQuote(defaultValue),
			),
		)
	}

	for _, envEntry := range dockerEntries {

		fmt.Fprintf(&sb, "%s \\\n", envEntry)
	}

	sb.WriteString("  your-image:latest\n")
	sb.WriteString("```\n")

	// ---------------------------------------------------------
	// Write the document
	// ---------------------------------------------------------

	if err := os.WriteFile(outputPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("failed to write Markdown documentation: %w", err)
	}

	return nil
}
