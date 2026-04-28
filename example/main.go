package main

import (
	"fmt"
	"log"
	"os"

	"github.com/smartfactory-kl/appgofig"
)

type Config struct {
	MyOwnSetting    int    `default:"42" env:"MY_OWN_SETTING"`
	MyStringSetting string `default:"defaultStringSetting" env:"MY_STRING_SETTING" req:"true"`
}

var configDescriptions map[string]string = map[string]string{
	"MyOwnSetting":    "This is just a simple example description so this map is not empty",
	"MyStringSetting": "This is just a string setting that is empty but required.",
}

func printConfig(cfg *Config) {
	fmt.Println("Configuration Output:")
	if err := appgofig.VisitConfigEntries(cfg, func(entry appgofig.ConfigEntry) {
		fmt.Fprintf(os.Stdout, "%s=%s\n", entry.Key, entry.Value)
	}); err != nil {
		log.Fatal(err)
	}
}

func main() {
	cfg := &Config{}

	// Standard way of using it
	if err := appgofig.ReadConfig(cfg); err != nil {
		log.Fatal(err)
	}
	printConfig(cfg)

	// documentation helper functions
	appgofig.WriteToMarkdownFile(cfg, configDescriptions, "example/MarkdownExample.md")
	appgofig.WriteToYamlExampleFile(cfg, configDescriptions, "example/ConfigYamlExample.yaml")

	// showcasing all options
	nextCfg := &Config{}

	if err := appgofig.ReadConfig(
		nextCfg,
		appgofig.WithReadMode(appgofig.ReadModeEnvThenYaml),
		appgofig.WithYamlFile("example/my_yaml.yml"),
	); err != nil {
		log.Fatal(err)
	}

	printConfig(nextCfg)

	// showcasing map input
	testCfg := &Config{}

	if err := appgofig.ReadConfig(
		testCfg,
		appgofig.WithReadMode(appgofig.ReadModeMapInputOnly),
		appgofig.WithMapInput(map[string]string{
			"MyOwnSetting": "1000",
		}),
	); err != nil {
		log.Fatal(err)
	}

	printConfig(testCfg)
}
