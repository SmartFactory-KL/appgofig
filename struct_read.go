package appgofig

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// isValidConfigStruct checks wether cfg points to a non-nil struct.
func isValidConfigStruct(cfg any) error {
	if cfg == nil {
		return fmt.Errorf("Config cannot be nil")
	}

	v := reflect.ValueOf(cfg)

	if v.Kind() != reflect.Pointer {
		return fmt.Errorf("Config must be a pointer, instead got %s", v.Kind())
	}

	if v.IsNil() {
		return fmt.Errorf("Config must not be a nil pointer")
	}

	if v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("Config must point to a struct, instead got %s", v.Elem().Type())
	}

	if err := onlyContainsSupportedTypes(cfg); err != nil {
		return fmt.Errorf("Config contains invalid types: %w", err)
	}

	return nil
}

// onlyContainsSupportedTypes checks if only supported data types are present within targtConfig
// if not, if returns an error describing the first non-valid field name
// It expects cfg to be a non-nil pointer to a non-nil struct.
func onlyContainsSupportedTypes(cfg any) error {
	t := reflect.TypeOf(cfg).Elem()

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

// readConfigDefaults reads the config struct and creates a map of AppConfigEntry based on its
// fields and tags. It expects cfg to be a non-nil pointer to a non-nil struct containing.
func readConfigDefaults(cfg any) map[string]*AppConfigEntry {
	t := reflect.TypeOf(cfg).Elem()

	entryMap := make(map[string]*AppConfigEntry, t.NumField())

	for k := 0; k < t.NumField(); k++ {
		field := t.Field(k)

		defaultValue := field.Tag.Get("default")
		envKey := field.Tag.Get("env")

		entry := &AppConfigEntry{
			Key:   field.Name,
			Value: defaultValue,

			DefaultValue: defaultValue,

			IsRequired: isRequiredField(field),
			IsMasked:   isMaskedField(field),

			EnvironmentKey: envKey,
		}

		entryMap[field.Name] = entry
	}

	return entryMap
}

// checkRequiredFields checks all entries for len(val) > 0 when isRequired is set
func checkRequiredFields(cfg map[string]*AppConfigEntry) error {
	missingRequiredKeys := []string{}

	for cfgKey, cfgEntry := range cfg {
		if !cfgEntry.IsRequired {
			continue
		}

		if len(cfgEntry.Value) == 0 {
			missingRequiredKeys = append(missingRequiredKeys, cfgKey)
		}
	}

	if len(missingRequiredKeys) > 0 {
		return fmt.Errorf("required keys are empty: %s", strings.Join(missingRequiredKeys, ","))
	}

	return nil
}

// hasOneBoolTagSet returns true if any of the tagsToCheck contains a string value that strconv.ParseBool would parse to true.
// returns false otherwise. Priority is: first ok tag in tagsToCheck gets the win.
func hasOneBoolTagSet(field reflect.StructField, tagsToCheck []string) bool {
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

// readStringFromValue returns a string representation of supported values
func readStringFromValue(fieldVal reflect.Value) string {
	switch fieldVal.Kind() {
	case reflect.String:
		return fieldVal.String()
	case reflect.Int:
		return strconv.FormatInt(fieldVal.Int(), 10)
	case reflect.Float64:
		return strconv.FormatFloat(fieldVal.Float(), 'f', -1, 64)
	case reflect.Bool:
		return strconv.FormatBool(fieldVal.Bool())
	default:
		return " - unsupported type " + fieldVal.Kind().String() + " - "
	}
}

// isMaskedField returns true if masked or mask is set
func isMaskedField(field reflect.StructField) bool {
	return hasOneBoolTagSet(field, []string{"masked", "mask"})
}

// isRequiredField returns true if required, req or require is set
func isRequiredField(field reflect.StructField) bool {
	return hasOneBoolTagSet(field, []string{"required", "req", "require"})
}
