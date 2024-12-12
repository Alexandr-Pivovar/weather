package main

import (
	"context"
	_ "embed"
	"log"

	"gopkg.in/yaml.v3"

	"weather/apis/geocoding"
	"weather/apis/weatherapi"
	"weather/apis/weatherbit"
	"weather/cli"
	"weather/manager"
)

//go:embed config.yaml
var configRaw []byte

func main() {
	ctx := context.Background()

	config := make(map[string]interface{})

	err := yaml.Unmarshal(configRaw, &config)
	if err != nil {
		panic(err)
	}

	weatherApi := weatherapi.New(config)
	weatherBit := weatherbit.New(config)

	weatherManager := manager.New()
	weatherManager.SetGeocoding(geocoding.New(config))
	weatherManager.RegisterAPI(weatherBit, weatherApi)

	cmd, err := cli.New(weatherManager)
	if err != nil {
		log.Printf("new cli: %s\n", err)
	}

	if err = cmd.ExecuteContext(ctx); err != nil {
		log.Printf("exec: %s\n", err)
	}
}
