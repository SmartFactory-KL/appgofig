package appgofig

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"go.yaml.in/yaml/v3"
)

// applyDefaultsToConfig uses applyStringMapToConfig to apply the default string inputs to targetConfig
func applyDefaultsToConfig(targetConfig any) error {
	// read defaults from the type into a map
	t := reflect.TypeOf(targetConfig).Elem()
	defaultsMap := make(map[string]string)

	for k := 0; k < t.NumField(); k++ {
		field := t.Field(k)
		defaultsMap[field.Name] = strings.TrimSpace(field.Tag.Get("default"))
	}

	if err := applyStringMapToConfig(targetConfig, defaultsMap); err != nil {
		return fmt.Errorf("unable to apply default values: %w", err)
	}

	return nil
}

// applyEnvironmentToConfig applies environment values to targetConfig while loading .env files first
func applyEnvironmentToConfig(targetConfig any) error {
	// load .env
	// error is ignored on purpose, as not having .env is not an issue
	godotenv.Load()

	// gather environment map
	envMap := make(map[string]string)

	t := reflect.TypeOf(targetConfig).Elem()
	for k := 0; k < t.NumField(); k++ {
		field := t.Field(k)
		fieldEnv, hasEnv := field.Tag.Lookup("env")

		keyToUse := field.Name
		if hasEnv && len(strings.TrimSpace(fieldEnv)) > 0 {
			keyToUse = fieldEnv
		}

		envVal, hasEnvVal := os.LookupEnv(keyToUse)

		// although the envKey is used to lookup the value,
		// the envMap needs the actual field.Name here as that is used to
		// map it to the field name in the actual config struct
		if hasEnvVal {
			envMap[field.Name] = strings.TrimSpace(envVal)
		}
	}

	if err := applyStringMapToConfig(targetConfig, envMap); err != nil {
		return err
	}

	return nil
}

// applyYamlToConfig checks for (config/)config.y(a)ml files and applies the first one found to targetConfig
func applyYamlToConfig(targetConfig any, gofigOptions *AppGofigOptions) error {
	yamlFilePath := ""
	if gofigOptions.YamlFileRequested {
		yamlFilePath = gofigOptions.YamlFilePath
	} else {
		// check for a config.yml or config.yaml in root directory or within a config folder
		// any value here overwrites the rest
		defaultYamlPaths := []string{"config.yml", "config.yaml", "config/config.yml", "config/config.yaml"}
		for _, path := range defaultYamlPaths {
			if _, err := os.Stat(path); err == nil {
				yamlFilePath = path
				break
			}
		}
	}

	if len(yamlFilePath) == 0 {
		return nil
	}

	data, err := os.ReadFile(filepath.Clean(yamlFilePath))
	if err != nil {
		return err
	}

	yamlMap := make(map[string]string)

	if err := yaml.Unmarshal(data, &yamlMap); err != nil {
		return err
	}

	if err := applyStringMapToConfig(targetConfig, yamlMap); err != nil {
		return err
	}

	return nil
}

// applyStringMapToConfig sets values on targetConfig based on a string map where fieldName == stringMapKey. Non-existing keys are ignored.
func applyStringMapToConfig(targetConfig any, stringValueMap map[string]string) error {
	// iterate over targetConfig while applying the string values converted to the actual target type

	v := reflect.ValueOf(targetConfig).Elem()
	t := v.Type()

	for k := 0; k < v.NumField(); k++ {
		field := t.Field(k)
		fieldVal := v.Field(k)

		// ignore non-existent keys
		if stringInput, ok := stringValueMap[field.Name]; !ok {
			continue
		} else {
			if err := applyStringToValue(field, fieldVal, strings.TrimSpace(stringInput)); err != nil {
				return fmt.Errorf("unable to write value %s to field %s : %w", stringInput, field.Name, err)
			}
		}
	}

	return nil
}

// applyStringToValue takes an input string and tries to convert it to the supported target types
func applyStringToValue(field reflect.StructField, fieldVal reflect.Value, input string) error {
	switch field.Type.Kind() {
	case reflect.String:
		fieldVal.SetString(input)
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(input)
		if err != nil {
			return fmt.Errorf("cannot use %s as bool: %w", input, err)
		}

		fieldVal.SetBool(boolVal)
	case reflect.Int:
		intVal, err := strconv.ParseInt(input, 0, 64)
		if err != nil {
			return fmt.Errorf("cannot use %s as int: %w", input, err)
		}

		fieldVal.SetInt(intVal)
	case reflect.Float64:
		floatVal, err := strconv.ParseFloat(input, 64)
		if err != nil {
			return fmt.Errorf("cannot use %s as float: %w", input, err)
		}

		fieldVal.SetFloat(floatVal)
	default:
		return fmt.Errorf("field %s has invalid config type %s which is not supported", field.Name, field.Type.Kind())
	}
	return nil
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
