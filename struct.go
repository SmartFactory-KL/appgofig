package appgofig

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// checkConfigStruct checks wether cfg points to a non-nil struct.
func checkConfigStruct(cfg any) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	v := reflect.ValueOf(cfg)

	if v.Kind() != reflect.Pointer {
		return fmt.Errorf("config must be a pointer, instead got %s", v.Kind())
	}

	if v.IsNil() {
		return fmt.Errorf("config must not be a nil pointer")
	}

	if v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("config must point to a struct, instead got %s", v.Elem().Type())
	}

	if err := onlyContainsExportedFields(cfg); err != nil {
		return fmt.Errorf("config contains invalid fields: %w", err)
	}

	if err := onlyContainsSupportedTypes(cfg); err != nil {
		return fmt.Errorf("config contains invalid types: %w", err)
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

// onlyContainsExportedFields checks wether unexported fields are present in the struct. They are not allowed.
func onlyContainsExportedFields(cfg any) error {
	t := reflect.TypeOf(cfg).Elem()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if field.PkgPath != "" {
			return fmt.Errorf("field %s must be exported", field.Name)
		}
	}

	return nil
}

// readConfigDefaults reads the config struct and creates a map of AppConfigEntry based on its
// fields and tags. It expects cfg to be a non-nil pointer to a non-nil struct.
func readConfigDefaults(cfg any) map[string]*AppConfigEntry {
	t := reflect.TypeOf(cfg).Elem()

	entryMap := make(map[string]*AppConfigEntry, t.NumField())

	for k := 0; k < t.NumField(); k++ {
		field := t.Field(k)

		defaultValue := strings.TrimSpace(field.Tag.Get("default"))
		envKey := strings.TrimSpace(field.Tag.Get("env"))

		entry := &AppConfigEntry{
			Key:   field.Name,
			Value: defaultValue,

			ValueType: field.Type.Kind(),

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

// applyEntryToValue tries to set fieldVals value by converting cfgEntry.value to the desired type.
func applyEntryToValue(field reflect.StructField, fieldVal reflect.Value, cfgEntry *AppConfigEntry) error {
	if !fieldVal.CanSet() {
		return fmt.Errorf("field %s cannot be set", field.Name)
	}

	switch field.Type.Kind() {
	case reflect.String:
		fieldVal.SetString(cfgEntry.Value)
	case reflect.Bool:
		var boolVal bool
		var err error
		if len(cfgEntry.Value) == 0 {
			// special case: Not having any value will be interpreted as "flag not set"
			// without ParseBool since that would fail but empty value as false could be reasonable
			boolVal = false
		} else {
			boolVal, err = strconv.ParseBool(cfgEntry.Value)
			if err != nil {
				return fmt.Errorf("cannot use %s as bool: %w", cfgEntry.Value, err)
			}
		}

		fieldVal.SetBool(boolVal)
	case reflect.Int:
		// base 0 means: Infer base from string input
		intVal, err := strconv.ParseInt(cfgEntry.Value, 0, 64)
		if err != nil {
			return fmt.Errorf("cannot use %s as int: %w", cfgEntry.Value, err)
		}

		fieldVal.SetInt(intVal)
	case reflect.Float64:
		floatVal, err := strconv.ParseFloat(cfgEntry.Value, 64)
		if err != nil {
			return fmt.Errorf("cannot use %s as float64: %w", cfgEntry.Value, err)
		}

		fieldVal.SetFloat(floatVal)
	default:
		return fmt.Errorf("unsupported type %s", field.Type.Kind())
	}

	return nil
}

// getConfigEntryKeys will return a list of strings that represent the order
// of keys within the cfg struct. Use this when iterating over the config sinces maps
// might not be consistent in their ordering
// Note: It expects cfg to be a non-nil pointer to a non-nil struct.
func getConfigEntryKeys(cfg any) []string {
	t := reflect.TypeOf(cfg).Elem()

	keys := make([]string, 0, t.NumField())

	for k := 0; k < t.NumField(); k++ {
		keys = append(keys, t.Field(k).Name)
	}

	return keys
}
