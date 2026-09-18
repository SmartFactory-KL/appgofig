package appgofig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// typing test
var _ AppGofigSource = (*yamlSrc)(nil)
var _ AppGofigSource = (*envSrc)(nil)

// common interface for all source loaders
type AppGofigSource interface {
	// Load will actually load from the source while passing the AppConfigEntries to read info like envKey from
	// It is expected that the returned map only contains keys that are also present within the config struct itself
	// so non-included keys must be filtered out.
	Load(map[string]*AppConfigEntry) (map[string]string, error)
}

// Source: Reading from a yaml file
type yamlSrc struct {
	filePath string
}

func SpecificYAMLSource(filePath string) AppGofigSource {
	return &yamlSrc{
		filePath: filePath,
	}
}

// Read from default yaml sources
func YAMLSource() AppGofigSource {
	return &yamlSrc{
		filePath: "",
	}
}

// Load will read the specified yaml file and return a map of values for which a key in cfgInfo exists.
func (src *yamlSrc) Load(cfgInfo map[string]*AppConfigEntry) (map[string]string, error) {
	var pathsToCheck []string

	if len(src.filePath) > 0 {
		pathsToCheck = append(pathsToCheck, src.filePath)
	} else {
		pathsToCheck = append(pathsToCheck,
			"config.yml",
			"config.yaml",
			"config/config.yml",
			"config/config.yaml",
		)
	}

	// read first existing file
	var yamlPath string
	for _, pathToCheck := range pathsToCheck {
		info, err := os.Stat(pathToCheck)
		if err == nil {
			if !info.IsDir() {
				yamlPath = pathToCheck
				break
			}
			continue
		}

		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("checking path %q failed: %w", pathToCheck, err)
		}
	}

	if len(strings.TrimSpace(yamlPath)) == 0 {
		// if a file was specified, it has to exist.
		// if the defaults were used and none of them was found, simply assume its default settings and return an empty map
		if len(src.filePath) > 0 {
			return nil, fmt.Errorf("YAML source cannot be read: file path cannot be empty or file not found for path: %s", yamlPath)
		} else {
			return map[string]string{}, nil
		}
	}

	yamlContent, err := os.ReadFile(filepath.Clean(yamlPath))
	if err != nil {
		return nil, fmt.Errorf("YAML source cannot be read: %w", err)
	}

	var yamlMap map[string]string
	if err := yaml.Unmarshal(yamlContent, &yamlMap); err != nil {
		return nil, fmt.Errorf("YAML source cannot be read: %w", err)
	}

	cfgMap := make(map[string]string, len(cfgInfo))
	for resultKey := range cfgInfo {
		yamlVal, ok := yamlMap[resultKey]
		if !ok {
			continue
		}

		cfgMap[resultKey] = strings.TrimSpace(yamlVal)
	}

	return cfgMap, nil
}

// Source: Reading from Environment
type envSrc struct {
	envPrefix string
}

func PrefixedEnvironmentSource(prefix string) AppGofigSource {
	return &envSrc{
		envPrefix: prefix,
	}
}

// Use the environment as source
func EnvironmentSource() AppGofigSource {
	return &envSrc{
		envPrefix: "",
	}
}

// Reads the environment using either EnvironmentKey or PREFIX_KEY as key
func (src *envSrc) Load(cfgInfo map[string]*AppConfigEntry) (map[string]string, error) {
	output := make(map[string]string, len(cfgInfo))

	for _, infoEntry := range cfgInfo {
		envKey := getEntryEnvKey(src.envPrefix, infoEntry)
		envVal, ok := os.LookupEnv(envKey)
		if !ok {
			continue
		}

		output[infoEntry.Key] = strings.TrimSpace(envVal)
	}

	return output, nil
}
