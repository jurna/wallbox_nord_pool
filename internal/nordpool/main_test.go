package nordpool

import (
	"math"
	"testing"
	"time"
)

func TestCalculatePrice(t *testing.T) {
	config := NordPoolConfig{
		MaxPrice:         0,
		Vat:              0,
		Timezone:         "Europe/Vilnius",
		TransmissionCost: TransmissionCostConfig{Day: 0.1, Night: 0.05, DayStartsAt: 7, NightStartsAt: 23, Timezone: "Etc/GMT-2"},
	}
	location, _ := time.LoadLocation(config.Timezone)
	tests := []struct {
		name        string
		currentTime time.Time
		wantPrice   float64
	}{
		{name: "Night", currentTime: time.Date(2023, 8, 30, 7, 0, 0, 0, location), wantPrice: 0.1},
		{name: "Day", currentTime: time.Date(2023, 8, 30, 23, 0, 0, 0, location), wantPrice: 0.15},
		{name: "Weekend", currentTime: time.Date(2023, 8, 26, 23, 0, 0, 0, location), wantPrice: 0.1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := calculatePrice(tt.currentTime, 50, config)
			if err != nil {
				t.Errorf("Got Error %s", err)
			}
			if math.Abs(p-tt.wantPrice) > 0.001 {
				t.Errorf("Got price %f, wanted %f", p, tt.wantPrice)
			}
		})
	}
}

func TestFindMinPrice(t *testing.T) {
	config := NordPoolConfig{
		MaxPrice:         0,
		ChargeTillHour:   8,
		Vat:              0,
		Timezone:         "Europe/Vilnius",
		TransmissionCost: TransmissionCostConfig{Day: 0.1, Night: 0.05, DayStartsAt: 7, NightStartsAt: 23, Timezone: "Etc/GMT-2"},
	}
	location, _ := time.LoadLocation(config.Timezone)
	tests := []struct {
		name      string
		prices    []Price
		wantPrice float64
	}{
		{name: "Empty", prices: []Price{}, wantPrice: math.MaxFloat64},
		{name: "Single", prices: []Price{{Timestamp: 1690840800, Price: 100}}, wantPrice: 0.15},
		{name: "First lower", prices: []Price{{Timestamp: 1690840800, Price: 100}, {Timestamp: 1690844400, Price: 200}}, wantPrice: 0.15},
		{name: "Second lower", prices: []Price{{Timestamp: 1690840800, Price: 200}, {Timestamp: 1690844400, Price: 100}}, wantPrice: 0.15},
		{name: "Three prices", prices: []Price{{Timestamp: 1690840800, Price: 200}, {Timestamp: 1690844400, Price: 100}, {Timestamp: 1690848000, Price: 200}}, wantPrice: 0.15},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := findMinPrice(config, tt.prices, time.Date(2023, 8, 1, 1, 0, 0, 0, location))
			if err != nil {
				t.Errorf("Got Error %s", err)
			}
			if math.Abs(p-tt.wantPrice) > 0.001 {
				t.Errorf("Got price %f, wanted %f", p, tt.wantPrice)
			}
		})
	}
}
