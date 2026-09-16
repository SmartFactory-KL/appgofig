package appgofig

import (
	"fmt"
	"reflect"
)

// ReadConfig takes your targetConfig struct, applies defaults and then applies values according to the readMode
// Using yamlFile, you can specify a yaml file to read from. If not specified, one of ./(config/)config.y(a)ml is used
func ReadConfig(cfg any, optionList ...AppGofigOption) error {
	// apply the options
	gofigOptions := &AppGofigOptions{
		Sources:   nil,
		Overrides: nil,
	}
	for _, opt := range optionList {
		opt(gofigOptions)
	}

	// check validitiy of cfg input
	if err := isValidConfigStruct(cfg); err != nil {
		return err
	}

	// read defaults first
	resultConfig := readConfigDefaults(cfg)

	// Load all sources in order
	sourceMaps := make([]map[string]string, len(gofigOptions.Sources))
	if len(gofigOptions.Sources) != 0 {
		// validate sources
		for _, src := range gofigOptions.Sources {
			srcMap, err := src.Load(resultConfig)
			if err != nil {
				return err
			}

			sourceMaps = append(sourceMaps, srcMap)
		}
	}

	// Apply all source maps to resultConfig
	for _, sourceMap := range sourceMaps {
		for cfgKey := range resultConfig {
			val, ok := sourceMap[cfgKey]
			if ok {
				resultConfig[cfgKey].Value = val
			}
		}
	}

	// Apply overrides at the end
	if len(gofigOptions.Overrides) > 0 {
		for cfgKey := range resultConfig {
			val, ok := gofigOptions.Overrides[cfgKey]
			if ok {
				resultConfig[cfgKey].Value = val
			}
		}
	}

	// check if all required keys are non-empty
	if err := checkRequiredFields(resultConfig); err != nil {
		return fmt.Errorf("missing required fields: %w", err)
	}

	// apply values to actual config struct and return it

	return nil
}

// VisitConfigEntries will run visit() once using AppConfigEntries but with an empty value.
// This is intended for creating documentation, but not for logging the actual configuration at startup for example.
// This can also be used with a non-initialized config
func VisitConfigEntries(cfg any, visit func(AppConfigEntry)) error {
	// check validitiy of cfg input
	if err := isValidConfigStruct(cfg); err != nil {
		return err
	}

	defaultValues := readConfigDefaults(cfg)

	for _, entry := range defaultValues {
		visit(AppConfigEntry{
			Key:            entry.Key,
			Value:          "",
			DefaultValue:   entry.DefaultValue,
			IsRequired:     entry.IsRequired,
			IsMasked:       entry.IsMasked,
			EnvironmentKey: entry.EnvironmentKey,
		})
	}

	return nil
}

// VisitConfigValues will run visit() once using AppConfigEntries with the actual value taken from cfg itself.
// Any values marked as "IsMasked" will be converted to "[Masked (len:x)]" with x being the string length
// This is intended to log out the actual configuration that an application started with and
// therefore requires an already initialized config struct to work
func VisitConfigValues(cfg any, visit func(AppConfigEntry)) error {
	// check validitiy of cfg input
	if err := isValidConfigStruct(cfg); err != nil {
		return err
	}

	if visit == nil {
		return fmt.Errorf("visit function cannot be nil")
	}

	cfgValues := reflect.ValueOf(cfg).Elem()
	defaultValues := readConfigDefaults(cfg)

	for _, entry := range defaultValues {
		field := cfgValues.FieldByName(entry.Key)

		value := readStringFromValue(field)

		if entry.IsMasked {
			value = fmt.Sprintf("[Masked (len: %d)]", len(value))
		}

		visit(AppConfigEntry{
			Key:            entry.Key,
			Value:          value,
			DefaultValue:   entry.DefaultValue,
			IsRequired:     entry.IsRequired,
			IsMasked:       entry.IsMasked,
			EnvironmentKey: entry.EnvironmentKey,
		})
	}

	return nil
}
