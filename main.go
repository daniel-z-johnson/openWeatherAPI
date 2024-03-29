package main

import (
	"context"
	"github.com/daniel-z-johnson/peronalWeatherSite/config"
	"github.com/daniel-z-johnson/peronalWeatherSite/models"
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
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
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
	weatherList, err := wa.GetListWeatherLocs()
	if err != nil {
		logger.Info(err.Error())
		panic(err)
	}
	logger.Info("it works, or at least no errors")
	for _, weather := range weatherList {
		logger.Info("Weather", "city", weather.Location.City, "Admin", weather.Location.AdminArea, "F", weather.Conditions.TempF)
	}
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
