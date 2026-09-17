package main

import (
	"github.com/smartfactory-kl/appgofig"
)

type ExampleConfig struct {
	MyOwnSetting    int    `default:"42" env:"MY_OWN_SETTING"`
	MyStringSetting string `default:"defaultStringSetting" env:"MY_STRING_SETTING" req:"true"`
}

var exampleCfgDescriptions map[string]string = map[string]string{
	"MyOwnSetting":    "This is just a simple example description so this map is not empty",
	"MyStringSetting": "This is just a string setting that is empty but required.",
}

func ExampleReadConfig() {
	cfg := ExampleConfig{}
	appgofig.ReadConfig(&cfg)
}

func ExampleCreateConfigDocumentation() {
	cfg := ExampleConfig{}
	appgofig.CreateConfigDocumentation(&cfg, exampleCfgDescriptions, "docs")
}
