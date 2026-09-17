package example_test

import (
	"log"

	"github.com/smartfactory-kl/appgofig"
)

type ExampleConfig struct {
	MyOwnSetting    int    `default:"42" env:"MY_OWN_SETTING"`
	MyStringSetting string `default:"StringInput" env:"MY_STRING_SETTING" req:"true"`
}

var exampleCfgDescriptions map[string]string = map[string]string{
	"MyOwnSetting":    "This is just a simple example description so this map is not empty",
	"MyStringSetting": "This is just a string setting that is empty but required.",
}

func ExampleReadConfig() {
	cfg := ExampleConfig{}
	err := appgofig.ReadConfig(&cfg, appgofig.WithSources(appgofig.EnvironmentSource()))
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleCreateConfigDocumentation() {
	cfg := ExampleConfig{}
	err := appgofig.CreateConfigDocumentation(&cfg, exampleCfgDescriptions, "docs")
	if err != nil {
		log.Fatal(err)
	}
}
