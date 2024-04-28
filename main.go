package main

import (
	"context"
	"github.com/daniel-z-johnson/peronalWeatherSite/config"
	"github.com/daniel-z-johnson/peronalWeatherSite/controllers"
	"github.com/daniel-z-johnson/peronalWeatherSite/models"
	"github.com/daniel-z-johnson/peronalWeatherSite/templates"
	"github.com/daniel-z-johnson/peronalWeatherSite/views"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"time"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitemigration"

	"log/slog"
	"os"
)

const (
	LocationsTableCreate = `CREATE TABLE LOCATIONS(
    			id INTEGER PRIMARY KEY AUTOINCREMENT,
    			key text, 
    			created_at datetime,
    			country text,
    			admin_area text,
    			city text,
    			country_code text,
    			admin_area_code text
				)`
	CurrentConditionsTableCreate = `CREATE TABLE CONDITIONS(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		locations_id integer,
		temp_c real,
		temp_f real,
		weather_type text,
		created_at datetime
	)`
)

func main() {
	logger := setUpLogger()
	logger = logger.With("application", "personal weather application")
	logger.Info("Application start")
	logger.Info(time.Now().Add(time.Hour * -2).Format(time.RFC3339))
	conn, err := connectDB(logger)
	if err != nil {
		// nothing can be done so give up and cause program to crash
		panic(err)
	}
	migrate(conn, logger)

	// quick testing delete later
	fileLoc := "config.json"
	conf, err := config.LoadConfig(&fileLoc)
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}
	wa := models.WeatherService(logger, conf, conn)
	// View
	weatherMainPage, err := views.ParseFS(logger, templates.FS, "central-layout.gohtml", "personalWeather.gohtml")
	if err != nil {
		logger.Error("Issue with parsing templates", "errMSG", err.Error())
		panic(err)
	}

	wc := &controllers.Weather{WeatherAPI: wa, Log: logger, PersonalWeather: weatherMainPage}
	r := chi.NewRouter()
	r.Get("/", wc.ShowCities)
	http.ListenAndServe(":1777", r)
}

func connectDB(logger *slog.Logger) (*sqlite.Conn, error) {
	conn, err := sqlite.OpenConn("w.db", sqlite.OpenReadWrite, sqlite.OpenCreate)
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}
	return conn, nil
}

func migrate(conn *sqlite.Conn, logger *slog.Logger) {
	schema := sqlitemigration.Schema{
		AppID: 0xb19b66b,
		Migrations: []string{
			LocationsTableCreate,
			CurrentConditionsTableCreate,
		},
	}
	err := sqlitemigration.Migrate(context.Background(), conn, schema)
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}
	logger.Info("Migration Success")
}

func setUpLogger() *slog.Logger {
	f1, err := os.OpenFile("logs/pw.log", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		logger.Error("Unable to open log file", "errMSG", err.Error())
		return logger
	}
	return slog.New(slog.NewJSONHandler(io.MultiWriter(os.Stdout, f1), nil))
}
