package appgofig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// typing test
var sourceList []AppGofigSource = []AppGofigSource{
	&yamlSrc{},
	&envSrc{},
}

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

// Read from a YAML source. If filePath is set, the first string path will be used, otherwise one of the defaults.
func YAMLSource(filePath ...string) *yamlSrc {
	srcPath := ""

	if len(filePath) > 0 {
		srcPath = strings.TrimSpace(filePath[0])
	}

	return &yamlSrc{
		filePath: srcPath,
	}
}

// Load will read the specified yaml file and return a map of values for which a key in cfgInfo exists.
func (src *yamlSrc) Load(cfgInfo map[string]*AppConfigEntry) (map[string]string, error) {
	var pathsToCheck []string

	if len(src.filePath) > 0 {
		pathsToCheck = append(pathsToCheck, src.filePath)
	} else {
		pathsToCheck = []string{
			"config.yml",
			"config.yaml",
			"config/config.yml",
			"config/config.yaml",
		}
	}

	// read first existing file
	var yamlPath string
	for _, pathToCheck := range pathsToCheck {
		if _, err := os.Stat(pathToCheck); err == nil {
			yamlPath = pathToCheck
			break
		}
	}

	if len(strings.TrimSpace(yamlPath)) == 0 {
		return nil, fmt.Errorf("YAML source cannot be read: file path cannot be empty")
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

		cfgMap[resultKey] = yamlVal
	}

	return cfgMap, nil
}

// Source: Reading from Environment
type envSrc struct {
	envPrefix string
}

// Use the environment as source
func EnvironmentSource(envPrefix string) *envSrc {
	return &envSrc{
		envPrefix: envPrefix,
	}
}

// Reads the environment using either EnvironmentKey or PREFIX_KEY as key
func (src *envSrc) Load(cfgInfo map[string]*AppConfigEntry) (map[string]string, error) {
	output := make(map[string]string, len(cfgInfo))

	for key, infoEntry := range cfgInfo {
		var envKey string

		if len(infoEntry.EnvironmentKey) > 0 {
			envKey = getEnvKey(src.envPrefix, infoEntry.EnvironmentKey)
		} else {
			envKey = getEnvKey(src.envPrefix, key)
		}

		envVal, ok := os.LookupEnv(envKey)
		if !ok {
			continue
		}

		output[envKey] = strings.TrimSpace(envVal)
	}

	return output, nil
}
