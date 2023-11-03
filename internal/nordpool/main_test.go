package nordpool

import (
	"math"
	"testing"
	"time"
)

func TestCalculatePrice(t *testing.T) {
	location, _ := time.LoadLocation("Europe/Vilnius")
	config := NordPoolConfig{
		MaxPrice:         0,
		Vat:              0,
		TransmissionCost: TransmissionCostConfig{Day: 0.1, Night: 0.05, DayStartsAt: 7, NightStartsAt: 23, TimeOffset: 2},
	}
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
