package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	WeatherAPI struct {
		Key string `json:"key"`
	} `json:"weatherAPI"`

	Zipcodes []struct {
		PostalCode  string `json:"postalCode"`
		CountryCode string `json:"countryCode"`
	} `json:"zipcodes"`
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
