package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	WeatherAPI struct {
		Key string `json:"key"`
	} `json:"weatherAPI"`

	Cities []struct {
		City        string `json:"City"`
		AdminArea   string `json:"adminArea"`
		CountryCode string `json:"countryCode"`
	} `json:"cities"`
}

func LoadConfig(fileLocation *string) (*Config, error) {
	f1, err := os.Open(*fileLocation)
	if err != nil {
		return nil, err
	}
	defer f1.Close()
	decoder := json.NewDecoder(f1)
	var conf Config
	decoder.Decode(&conf)
	return &conf, err
}
