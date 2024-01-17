package models

import (
	"fmt"
	"github.com/daniel-z-johnson/peronalWeatherSite/config"
	"log/slog"
	"net/url"
)

type GeoPoint struct {
	Zip     string  `json:"zip"`
	Name    string  `json:"name"`
	Lat     float64 `json:"lat"`
	Long    float64 `json:"long"`
	Country string  `json:"country"`
}

type WeatherAPI struct {
	config *config.Config
	logger *slog.Logger
}

func WeatherService(logger *slog.Logger, config *config.Config) *WeatherAPI {
	return &WeatherAPI{logger: logger, config: config}
}

func (wa *WeatherAPI) GetGeoPoints() ([]GeoPoint, error) {
	geoPoints := make([]GeoPoint, 0)
	for _, zip := range wa.config.Zipcodes {
		u, err := url.Parse("https://dataservice.accuweather.com/locations/v1/postalcodes/search")
		if zip.CountryCode != "" {
			u, err = url.Parse(
				fmt.Sprintf("https://dataservice.accuweather.com/locations/v1/postalcodes/%s/search", zip.CountryCode))
		}
		if err != nil {
			return nil, err
		}
		values := u.Query()
		values.Set("zip", zip.PostalCode)
		u.RawQuery = values.Encode()
		wa.logger.Info(u.String())
	}

	return geoPoints, nil
}
