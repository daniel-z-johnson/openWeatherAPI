package models

import (
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
		u, err := url.Parse("https://api.openweathermap.org/geo/1.0/zip")
		if err != nil {
			return nil, err
		}
		values := u.Query()
		values.Set("zip", zip)
		u.RawQuery = values.Encode()
		wa.logger.Info(u.String())
	}

	return geoPoints, nil
}
