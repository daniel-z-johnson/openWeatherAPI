package models

import (
	"encoding/json"
	"fmt"
	"github.com/daniel-z-johnson/peronalWeatherSite/config"
	"log/slog"
	"net/http"
	"net/url"
	"time"
	"zombiezen.com/go/sqlite"
)

type GeoLocation struct {
	Version    int32  `json:"Version"`
	Key        string `json:"Key"`
	Type       string `json:"Type"`
	ParentCity struct {
		Key           string `json:"Key"`
		LocalizedName string `json:"LocalizedName"`
		EnglishName   string `json:"EnglishName"`
	} `json:"ParentCity"`
	Region struct {
		EnglishName string `json:"EnglishName"`
	} `json:"Region"`
	Country struct {
		EnglishName string `json:"EnglishName"`
	} `json:"Country"`
	AdministrativeArea struct {
		EnglishName string `json:"EnglishName"`
	} `json:"AdministrativeArea"`
	SupplementalAdminAreas []struct {
		EnglishName string `json:"EnglishName"`
	} `json:"SupplementalAdminAreas"`
}

type Location struct {
	ID          int64
	PostalCode  string
	Key         string
	CreatedAt   time.Time
	Country     string
	AdminArea   string
	Name        string
	CountryCode string
}

type WeatherAPI struct {
	config *config.Config
	db     *sqlite.Conn
	logger *slog.Logger
}

func WeatherService(logger *slog.Logger, config *config.Config, db *sqlite.Conn) *WeatherAPI {
	return &WeatherAPI{logger: logger, config: config, db: db}
}

// use time.RFC3339
func (wa *WeatherAPI) GetGeoPointFromDb(countryCode, postalCode string) (*Location, error) {
	stmt, _, err := wa.db.PrepareTransient(`SELECT id, postal_code, key, created_at, country, admin_area, name, country_code FROM locations 
         WHERE country_code = ? AND postal_code = ? and CREATED_AT > ? ORDER BY CREATED_AT DESC`)
	if err != nil {
		wa.logger.Error(fmt.Sprintf("Error occured in WeaherAPI.GetGeoPoints with %s", err.Error()))
		return nil, err
	}
	stmt.BindText(1, countryCode)
	stmt.BindText(2, postalCode)
	stmt.BindText(3, time.Now().Add(time.Hour*-24).Format(time.RFC3339))
	defer stmt.Finalize()
	rowReturned, err := stmt.Step()
	if err != nil {
		wa.logger.Error(err.Error())
		return nil, err
	}
	if rowReturned {
		location := &Location{}
		location.ID = stmt.GetInt64("id")
		location.PostalCode = stmt.GetText("postal_code")
		location.Key = stmt.GetText("key")
		createdAt, err := time.Parse(time.RFC3339, stmt.GetText("created_at"))
		if err != nil {
			wa.logger.Error("Error setting location.CreatedAt value")
		} else {
			location.CreatedAt = createdAt
		}
		location.Country = stmt.GetText("country")
		location.AdminArea = stmt.GetText("admin_area")
		location.Name = stmt.GetText("name")
		location.CountryCode = stmt.GetText("country_code")
		// add a string method to locations for changing into string for logging, json maybe
		wa.logger.Info("entry found", "location", fmt.Sprintf("%+v", location))
		return location, nil
	}
	wa.logger.Info("no entries found in db", "countryCode", countryCode, "postalCode", postalCode)
	return nil, nil
}

func (wa *WeatherAPI) GetLocationFromAccu(countryCode, postalCode string) (*Location, error) {
	wa.logger.Info("Accessing Accu Weather API for Location key and data")
	location := &Location{}
	u, err := url.Parse("https://dataservice.accuweather.com/locations/v1/postalcodes/search")
	if countryCode != "" {
		u, err = url.Parse(
			fmt.Sprintf("https://dataservice.accuweather.com/locations/v1/postalcodes/%s/search", countryCode))
	}
	if err != nil {
		return nil, err
	}
	values := u.Query()
	values.Set("q", postalCode)
	values.Set("apikey", wa.config.WeatherAPI.Key)
	u.RawQuery = values.Encode()
	// API will return area of possible locations
	geoLoc := make([]*GeoLocation, 0)
	r, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	err = json.NewDecoder(r.Body).Decode(&geoLoc)
	if err != nil {
		return nil, err
	}
	// Just grab the first one, since this i s for personal use only at this
	// point it is ok to assume there will be at least one location
	geoPoint := geoLoc[0]
	location.PostalCode = postalCode
	location.Key = geoPoint.Key
	location.Country = geoPoint.Country.EnglishName
	location.AdminArea = geoPoint.AdministrativeArea.EnglishName
	location.Name = geoPoint.ParentCity.EnglishName
	location.CountryCode = countryCode

	if location.Name == "" && len(geoPoint.SupplementalAdminAreas) > 0 {
		location.Name = geoPoint.SupplementalAdminAreas[0].EnglishName
	}
	return location, nil
}

func (wa *WeatherAPI) saveLocation(location *Location) (*Location, error) {
	stmt, _, err := wa.db.PrepareTransient(`INSERT INTO locations 
    								(postal_code, key, created_at, country, admin_area, name, country_code) VALUES 
    								(          ?,   ?,          ?,       ?,          ?,    ?,            ?)`)
	if err != nil {
		wa.logger.Error(err.Error())
		return nil, err
	}
	stmt.BindText(1, location.PostalCode)
	stmt.BindText(2, location.Key)
	stmt.BindText(3, time.Now().Format(time.RFC3339))
	stmt.BindText(4, location.Country)
	stmt.BindText(5, location.AdminArea)
	stmt.BindText(6, location.Name)
	stmt.BindText(7, location.CountryCode)
	location.ID = wa.db.LastInsertRowID()
	_, err = stmt.Step()
	if err != nil {
		wa.logger.Error(err.Error())
		return nil, err
	}
	return location, nil
}

func (wa *WeatherAPI) GetLocation(countryCode, postalCode string) (*Location, error) {
	loc, err := wa.GetGeoPointFromDb(countryCode, postalCode)
	if err != nil {
		return nil, err
	}
	if loc == nil {
		loc, err = wa.GetLocationFromAccu(countryCode, postalCode)
		if err != nil {
			return nil, err
		}
		loc, _ = wa.saveLocation(loc)
	}
	return loc, nil
}

func (wa *WeatherAPI) GetLocations() ([]*Location, error) {
	locations := make([]*Location, 0)
	for _, v := range wa.config.Zipcodes {
		loc, err := wa.GetLocation(v.CountryCode, v.PostalCode)
		if err != nil {
			wa.logger.Info("Issue happened", "err", err.Error())
		}
		if loc != nil {
			locations = append(locations, loc)
		}
	}
	return locations, nil
}

func (wa *WeatherAPI) loadKeys() {

}
