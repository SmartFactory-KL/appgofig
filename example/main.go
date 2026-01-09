package main

import (
	"log"
	"os"

	"github.com/kzoltner/appgofig"
)

type Config struct {
	MyOwnSetting    int64  `default:"42" env:"MY_OWN_SETTING"`
	MyStringSetting string `default:"defaultStringSetting" env:"MY_STRING_SETTING" req:"true"`
}

var configDescriptions map[string]string = map[string]string{
	"MyOwnSetting":    "This is just a simple example description so this map is not empty",
	"MyStringSetting": "This is just a string setting that is empty but required.",
}

func main() {
	cfg := &Config{}

	// Standard way of using it
	if err := appgofig.ReadConfig(cfg); err != nil {
		log.Fatal(err)
	}
	appgofig.LogConfig(cfg, os.Stdout)

	// documentation helper functions
	appgofig.WriteToMarkdownFile(cfg, configDescriptions, "example/MarkdownExample.md")
	appgofig.WriteToYamlExampleFile(cfg, configDescriptions, "example/ConfigYamlExample.yaml")

	// showcasing all options
	nextCfg := &Config{}

	if err := appgofig.ReadConfig(
		nextCfg,
		appgofig.WithReadMode(appgofig.ReadModeEnvThenYaml),
		appgofig.WithYamlFile("example/my_yaml.yml"),
		appgofig.WithNewDefaults(map[string]string{
			"MyOwnSetting": "1000",
		}),
	); err != nil {
		log.Fatal(err)
	}

	appgofig.LogConfig(nextCfg, os.Stdout)
}
