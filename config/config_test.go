package config

import "testing"

func TestLoadConf(t *testing.T) {
	fileLoc := "config_test.json"
	c, e := LoadConfig(&fileLoc)
	if e != nil {
		t.Fatalf("loadConfig returned an error but shouldn't have: '%s'", e)
	}
	if c == nil {
		t.Fatal("config loaded is nil")
	}
	if c.Cities == nil {
		t.Fatal("Cities should not be nil")
	}
	if c.WeatherAPI.Key == "" {
		t.Error("weatherAPI key should not be empty")
	}

}
