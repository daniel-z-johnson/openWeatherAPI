package controllers

import (
	"github.com/daniel-z-johnson/peronalWeatherSite/models"
	"github.com/daniel-z-johnson/peronalWeatherSite/views"
	"log/slog"
	"net/http"
)

type Weather struct {
	WeatherAPI *models.WeatherAPI
	// TODO consider struct for all views in case more views are added
	PersonalWeather *views.Template
	Log             *slog.Logger
}

func (weather *Weather) ShowCities(w http.ResponseWriter, r *http.Request) {
	weatherLoc, err := weather.WeatherAPI.GetListWeatherLocs()
	if err != nil {
		weather.Log.Error("Unable to retrieve weather data for server gen page", "errMSG", err.Error())
		http.Error(w, "something went wrong, throw stuff at dev for terrible error msg until it is fixed", http.StatusInternalServerError)
	}
	weather.PersonalWeather.Execute(w, r, weatherLoc)
}
