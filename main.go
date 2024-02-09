package main

import (
	"context"
	"fmt"
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
    			postal_code text,
    			key text, 
    			created_at datetime,
    			country text,
    			admin_area text,
    			name text,
    			country_code text
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
	loc, err := wa.GetLocation("US", "78613")
	if err != nil {
		logger.Error(err.Error())
	}
	logger.Info(fmt.Sprintf("%+v", loc))
	wa.GetLocations()
	// geoPoints, err := wa.GetGeoPoints()
	//if err != nil {
	//	panic(err)
	//}
	//for _, geo := range geoPoints {
	//	fmt.Printf("%+v\n", geo)
	//}
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
		},
	}
	err := sqlitemigration.Migrate(context.Background(), conn, schema)
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}
	logger.Info("Migration Success")
}
