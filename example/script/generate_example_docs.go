//go:build ignore

package main

import (
	"log"

	"github.com/smartfactory-kl/appgofig"
)

type ExampleConfig struct {
	StringValue string  `default:"DefaultStringValue"`
	IntValue    int     `default:"100"`
	FloatValue  float64 `default:"3.141"`
	BoolValue   bool    `default:"true"`

	MaskedValue   float64 `default:"3.141" masked:"true"`
	RequiredValue string  `default:"MyRequiredValue" required:"true"`
}

var exampleCfgDescriptions map[string]string = map[string]string{
	"StringValue":   "An example for a string value",
	"IntValue":      "An example for an int value",
	"FloatValue":    "An example for a float value",
	"BoolValue":     "An example for a boolean value",
	"MaskedValue":   "An example for a masked value",
	"RequiredValue": "An example for a required value",
}

func main() {
	cfg := ExampleConfig{}

	if err := appgofig.CreateConfigDocumentation(
		&cfg,
		exampleCfgDescriptions,
		"MY_APP_PREFIX",
		"example/docs",
	); err != nil {
		log.Fatal(err)
	}
}
