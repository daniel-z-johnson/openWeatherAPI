package main

import (
	"github.com/daniel-z-johnson/peronalWeatherSite/config"
	"github.com/daniel-z-johnson/peronalWeatherSite/models"
	"log/slog"
	"os"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger = logger.With("application", "personal weather application")
	logger.Info("Application start")

	// quick testing delete later
	fileLoc := "config.json"
	conf, err := config.LoadConfig(&fileLoc)
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}
	wa := models.WeatherService(logger, conf)
	wa.GetGeoPoints()

}
