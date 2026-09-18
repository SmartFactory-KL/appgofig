package example_test

import (
	"fmt"
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

func ExampleReadConfig() {
	// This will simply apply the default values from the struct
	// as no sources are provided
	cfg, err := appgofig.ReadConfig[ExampleConfig]()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(cfg)
	// Output:
	// &{DefaultStringValue 100 3.141 true 3.141 MyRequiredValue}
}

func ExampleReadConfig_withSources() {
	// One standard way of reading configuration might be
	// reading YAML first with ENV overwriting it
	cfg, err := appgofig.ReadConfig[ExampleConfig](
		appgofig.WithSources(
			appgofig.YAMLSource(),
			appgofig.EnvironmentSource(),
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(cfg)
}

// This is how config documentation would be created
func ExampleCreateConfigDocumentation() {
	cfg := ExampleConfig{}
	err := appgofig.CreateConfigDocumentation(&cfg, exampleCfgDescriptions, "MY_APP_PREFIX", "docs")
	if err != nil {
		log.Fatal(err)
	}
}
