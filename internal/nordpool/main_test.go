package nordpool

import (
	"math"
	"testing"
	"time"
)

func TestCalculatePriceNight(t *testing.T) {
	location, _ := time.LoadLocation("Europe/Vilnius")
	currentTime := time.Date(2023, 8, 30, 7, 0, 0, 0, location)
	config := NordPoolConfig{
		MaxPrice:         0,
		Vat:              0,
		TransmissionCost: TransmissionCostConfig{Day: 0.1, Night: 0.05, DayStartsAt: 7, NightStartsAt: 23, Timezone: "Etc/GMT-2"},
	}

	p, err := calculatePrice(currentTime, 50, config)
	if err != nil {
		t.Fatalf("Got Error %s", err)
	}
	if math.Abs(p-0.1) > 0.001 {
		t.Fatalf("Got price %f", p)
	}

}

func TestCalculatePriceDay(t *testing.T) {
	location, _ := time.LoadLocation("Europe/Vilnius")
	currentTime := time.Date(2023, 8, 30, 23, 0, 0, 0, location)
	config := NordPoolConfig{
		MaxPrice:         0,
		Vat:              0,
		TransmissionCost: TransmissionCostConfig{Day: 0.1, Night: 0.05, DayStartsAt: 7, NightStartsAt: 23, Timezone: "Etc/GMT-2"},
	}

	p, err := calculatePrice(currentTime, 50, config)
	if err != nil {
		t.Fatalf("Got Error %s", err)
	}
	if math.Abs(p-0.15) > 0.001 {
		t.Fatalf("Got price %f", p)
	}

}

func TestCalculatePriceWeekend(t *testing.T) {
	location, _ := time.LoadLocation("Europe/Vilnius")
	currentTime := time.Date(2023, 8, 26, 23, 0, 0, 0, location)
	config := NordPoolConfig{
		MaxPrice:         0,
		Vat:              0,
		TransmissionCost: TransmissionCostConfig{Day: 0.1, Night: 0.05, DayStartsAt: 7, NightStartsAt: 23, Timezone: "Etc/GMT-2"},
	}

	p, err := calculatePrice(currentTime, 50, config)
	if err != nil {
		t.Fatalf("Got Error %s", err)
	}
	if math.Abs(p-0.1) > 0.001 {
		t.Fatalf("Got price %f", p)
	}

}
