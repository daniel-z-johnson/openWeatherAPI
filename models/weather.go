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
	Version     int32  `json:"Version"`
	Key         string `json:"Key"`
	Type        string `json:"Type"`
	EnglishName string `json:"EnglishName"`
	ParentCity  struct {
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
	ID            int64
	Key           string
	CreatedAt     time.Time
	Country       string
	AdminArea     string
	City          string
	CountryCode   string
	AdminAreaCode string
}

type Conditions struct {
	ID          int64
	LocationsID int64
	TempC       float64
	TempF       float64
	WeatherText string
	CreatedAt   time.Time
}

type WeatherAPI struct {
	config *config.Config
	db     *sqlite.Conn
	logger *slog.Logger
}

type CurrentConditions struct {
	WeatherText string `json:"WeatherText"`
	Temperature struct {
		Metric struct {
			Value float64 `json:"Value"`
		} `json:"Metric"`
		Imperial struct {
			Value float64 `json:"value"`
		} `json:"Imperial"`
	} `json:"Temperature"`
}

type WeatherLoc struct {
	Location   *Location
	Conditions *Conditions
}

func WeatherService(logger *slog.Logger, config *config.Config, db *sqlite.Conn) *WeatherAPI {
	return &WeatherAPI{logger: logger, config: config, db: db}
}

func (wa *WeatherAPI) GetGeoPointFromDb(countryCode, adminArea, city string) ([]*Location, error) {
	locations := make([]*Location, 0)
	query := `SELECT id, key, created_at, country, admin_area, city, country_code, admin_area_code FROM locations WHERE country_code = ? AND city = ? AND admin_area_code = ? AND CREATED_AT > ? ORDER BY CREATED_AT DESC`
	if adminArea == "" {
		query = `SELECT id, key, created_at, country, admin_area, city, country_code, admin_area_code FROM locations 
         WHERE country_code = ? AND city = ? AND CREATED_AT > ? ORDER BY CREATED_AT DESC`
	}

	stmt, _, err := wa.db.PrepareTransient(query)
	if err != nil {
		wa.logger.Error(fmt.Sprintf("Error occured in WeaherAPI.GetGeoPoints with %s", err.Error()))
		return nil, err
	}
	if adminArea != "" {
		stmt.BindText(1, countryCode)
		stmt.BindText(2, city)
		stmt.BindText(3, adminArea)
		stmt.BindText(4, time.Now().Add(time.Hour*-24).Format(time.RFC3339))
	} else {
		stmt.BindText(1, countryCode)
		stmt.BindText(2, city)
		stmt.BindText(3, time.Now().Add(time.Hour*-24).Format(time.RFC3339))
	}
	defer stmt.Finalize()

	for {
		rowReturned, err := stmt.Step()
		if err != nil {
			wa.logger.Error(err.Error())
			return nil, err
		}
		if rowReturned {
			location := &Location{}
			location.ID = stmt.GetInt64("id")
			location.Key = stmt.GetText("key")
			createdAt, err := time.Parse(time.RFC3339, stmt.GetText("created_at"))
			if err != nil {
				wa.logger.Error("Error setting location.CreatedAt value")
			} else {
				location.CreatedAt = createdAt
			}
			location.Country = stmt.GetText("country")
			location.AdminArea = stmt.GetText("admin_area")
			location.City = stmt.GetText("city")
			location.CountryCode = stmt.GetText("country_code")
			location.AdminAreaCode = stmt.GetText("admin_area_code")
			// add a string method to locations for changing into string for logging, json maybe
			wa.logger.Info("entry found", "location", fmt.Sprintf("%+v", location))
			locations = append(locations, location)
		}
		if !rowReturned {
			break
		}
	}
	if len(locations) == 0 {
		wa.logger.Info("no entries found in db", "countryCode", countryCode, "city", city)
		return nil, nil
	}
	return locations, err
}

func (wa *WeatherAPI) GetLocationFromAccu(countryCode, adminArea, city string) ([]*Location, error) {
	locations := make([]*Location, 0)
	wa.logger.Info("Accessing Accu Weather API for Location key and data")
	location := &Location{}
	u, err := url.Parse(fmt.Sprintf("http://dataservice.accuweather.com/locations/v1/cities/%s/search", countryCode))
	if adminArea != "" {
		u, err = url.Parse(
			fmt.Sprintf("http://dataservice.accuweather.com/locations/v1/cities/%s/%s/search", countryCode, adminArea))
	}
	if err != nil {
		wa.logger.Error("Issue with parsing error", "url", u.String())
		return nil, err
	}
	values := u.Query()
	values.Set("q", city)
	values.Set("apikey", wa.config.WeatherAPI.Key)
	u.RawQuery = values.Encode()
	// API will return array of possible locations
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
	for _, geoPoint := range geoLoc {
		location.Key = geoPoint.Key
		location.Country = geoPoint.Country.EnglishName
		location.AdminArea = geoPoint.AdministrativeArea.EnglishName
		location.City = geoPoint.ParentCity.EnglishName
		location.CountryCode = countryCode
		location.AdminAreaCode = adminArea

		if location.City == "" {
			location.City = geoPoint.EnglishName
		}

		if location.City == "" && len(geoPoint.SupplementalAdminAreas) > 0 {
			location.City = geoPoint.SupplementalAdminAreas[0].EnglishName
		}
		locations = append(locations, location)
	}
	return locations, nil
}

func (wa *WeatherAPI) saveLocation(locations []*Location) ([]*Location, error) {
	for _, location := range locations {
		stmt, _, err := wa.db.PrepareTransient(`INSERT INTO locations 
    								(key, created_at, country, admin_area, city, country_code, admin_area_code) VALUES 
    								(  ?,          ?,       ?,          ?,    ?,            ?,               ?)`)
		if err != nil {
			wa.logger.Error(err.Error())
			return nil, err
		}
		stmt.BindText(1, location.Key)
		stmt.BindText(2, time.Now().Format(time.RFC3339))
		stmt.BindText(3, location.Country)
		stmt.BindText(4, location.AdminArea)
		stmt.BindText(5, location.City)
		stmt.BindText(6, location.CountryCode)
		stmt.BindText(7, location.AdminAreaCode)
		_, err = stmt.Step()
		if err != nil {
			wa.logger.Error(err.Error())
			return nil, err
		}
		defer stmt.Finalize()
		location.ID = wa.db.LastInsertRowID()
	}

	return locations, nil
}

func (wa *WeatherAPI) GetLocation(countryCode, adminArea, city string) ([]*Location, error) {
	loc, err := wa.GetGeoPointFromDb(countryCode, adminArea, city)
	if err != nil {
		return nil, err
	}
	if loc == nil {
		loc, err = wa.GetLocationFromAccu(countryCode, adminArea, city)
		if err != nil {
			return nil, err
		}
		loc, _ = wa.saveLocation(loc)
	}
	return loc, nil
}

func (wa *WeatherAPI) GetCurrentCondition(locationID int64, key string) (*Conditions, error) {
	condition, err := wa.GetCurrentConditionFromDB(locationID)
	if err != nil {
		return nil, err
	}
	if condition == nil {
		condition, err = wa.GetCurrentConditionFromAccu(locationID, key)
		if err != nil {
			return nil, err
		}
		condition, _ = wa.SaveCurrentConditions(condition)
	}
	return condition, nil
}

func (wa *WeatherAPI) GetCurrentConditionFromAccu(locationID int64, key string) (*Conditions, error) {
	u, err := url.Parse("https://dataservice.accuweather.com/currentconditions/v1/" + key)
	conditions := &Conditions{}
	if err != nil {
		return nil, err
	}
	values := u.Query()
	values.Set("apikey", wa.config.WeatherAPI.Key)
	u.RawQuery = values.Encode()
	// accu returns an array of conditions
	currentConditions := make([]*CurrentConditions, 0)
	r, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	err = json.NewDecoder(r.Body).Decode(&currentConditions)
	if err != nil {
		return nil, err
	}
	// Just grab the first one, since this is for personal use only at this
	// point it is ok to assume there will be at least one location
	currentCondition := currentConditions[0]
	conditions.LocationsID = locationID
	conditions.WeatherText = currentCondition.WeatherText
	conditions.TempC = currentCondition.Temperature.Metric.Value
	conditions.TempF = currentCondition.Temperature.Imperial.Value
	return conditions, nil
}

func (wa *WeatherAPI) SaveCurrentConditions(conditions *Conditions) (*Conditions, error) {
	stmt, _, err := wa.db.PrepareTransient(`INSERT INTO CONDITIONS 
    								(locations_id, temp_c, temp_f, weather_type, created_at) VALUES 
                                     (          ?,      ?,      ?,            ?,           ?)`)
	if err != nil {
		wa.logger.Error(err.Error())
		return nil, err
	}
	stmt.BindInt64(1, conditions.LocationsID)
	stmt.BindFloat(2, conditions.TempC)
	stmt.BindFloat(3, conditions.TempF)
	stmt.BindText(4, conditions.WeatherText)
	stmt.BindText(5, time.Now().Format(time.RFC3339))
	_, err = stmt.Step()
	if err != nil {
		wa.logger.Error(err.Error())
		return nil, err
	}
	conditions.ID = wa.db.LastInsertRowID()
	return conditions, nil
}

func (wa *WeatherAPI) GetCurrentConditionFromDB(locationID int64) (*Conditions, error) {
	stmt, _, err := wa.db.PrepareTransient(`SELECT id, locations_id, temp_c, temp_f, weather_type, created_at FROM CONDITIONS
													WHERE locations_id = ? AND created_at > ? ORDER BY created_at DESC`)
	if err != nil {
		wa.logger.Error("Unable to process SQL statement for CONDICTIONS", "err", err.Error())
		return nil, err
	}

	stmt.BindInt64(1, locationID)
	stmt.BindText(2, time.Now().Add(-1*time.Hour).Format(time.RFC3339))
	rowReturn, err := stmt.Step()
	if err != nil {
		wa.logger.Error("Error executing CONDITIONS sql statement", "err", err.Error())
		return nil, err
	}

	if rowReturn {
		conditions := &Conditions{}
		conditions.ID = stmt.GetInt64("id")
		conditions.LocationsID = stmt.GetInt64("locations_id")
		conditions.TempF = stmt.GetFloat("temp_f")
		conditions.TempC = stmt.GetFloat("temp_c")
		conditions.WeatherText = stmt.GetText("weather_type")
		createdAt, err := time.Parse(time.RFC3339, stmt.GetText("created_at"))
		if err != nil {
			wa.logger.Error("Error setting location.CreatedAt value")
		} else {
			conditions.CreatedAt = createdAt
		}
		return conditions, nil

	}
	wa.logger.Info("No rows return for conditions", "locationID", locationID)
	return nil, nil
}

func (wa *WeatherAPI) GetListWeatherLocs() ([]*WeatherLoc, error) {
	weatherLocList := make([]*WeatherLoc, 0)
	for _, city := range wa.config.Cities {
		locs, err := wa.GetLocation(city.CountryCode, city.AdminArea, city.City)
		if err != nil {
			return nil, err
		}
		for _, loc := range locs {
			condition, err := wa.GetCurrentCondition(loc.ID, loc.Key)
			if err != nil {
				return nil, err
			}
			weatherLocList = append(weatherLocList, &WeatherLoc{Location: loc, Conditions: condition})
		}
	}
	return weatherLocList, nil
}
