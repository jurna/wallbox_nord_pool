package nordpool

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Price struct {
	Timestamp int64   `json:"timestamp"`
	Price     float64 `json:"price"`
}
type Prices struct {
	Success bool `json:"success"`
	Data    struct {
		Ee []Price `json:"ee"`
		Fi []Price `json:"fi"`
		Lv []Price `json:"lv"`
		Lt []Price `json:"lt"`
	} `json:"data"`
}

type TransmissionCostConfig struct {
	Day           float64 `yaml:"day"`
	Night         float64 `yaml:"night"`
	DayStartsAt   int     `yaml:"day-starts-at"`
	NightStartsAt int     `yaml:"night-starts-at"`
	Timezone      string  `yaml:"timezone"`
}

type NordPoolConfig struct {
	MaxPrice         float64                `yaml:"max-price"`
	Vat              float64                `yaml:"vat"`
	TransmissionCost TransmissionCostConfig `yaml:"transmission-cost"`
}

type PriceStatus string

const (
	PriceGood   PriceStatus = "PriceGood"
	PriceTooBig             = "PriceTooBig"
)

var (
	errPricesFileDoesNotExist = errors.New("prices file does not exist")
	errPriceNotFound          = errors.New("price not found")
)

func GetPrice(date time.Time, config NordPoolConfig) (price float64, err error) {
	prices, err := readPrices(date)
	if err != nil {
		if !errors.Is(err, errPricesFileDoesNotExist) {
			return
		}
		prices, err = fetchDates(date)
		if err != nil {
			return
		}
	}
	poolPrice, err := findPrice(prices.Data.Lt, date)
	price, err = calculatePrice(date, poolPrice, config)
	return
}

func calculatePrice(date time.Time, poolPrice float64, config NordPoolConfig) (price float64, err error) {
	costConfig := config.TransmissionCost
	location, err := time.LoadLocation(costConfig.Timezone)
	if err != nil {
		return
	}
	transmissionDate := date.In(location)
	workday := transmissionDate.Weekday() != time.Saturday && transmissionDate.Weekday() != time.Sunday

	if workday && transmissionDate.Hour() >= costConfig.DayStartsAt && transmissionDate.Hour() < costConfig.NightStartsAt {
		price = (poolPrice/1000)*(1+config.Vat) + costConfig.Day
	} else {
		price = (poolPrice/1000)*(1+config.Vat) + costConfig.Night
	}
	return
}

func findPrice(prices []Price, date time.Time) (price float64, err error) {
	timestamp := date.Truncate(time.Hour).Unix()
	for _, p := range prices {
		if p.Timestamp == timestamp {
			price = p.Price
			return
		}
	}
	return price, fmt.Errorf("%d : %w", timestamp, errPriceNotFound)
}

func fetchDates(date time.Time) (prices Prices, err error) {
	req, err := http.NewRequest("GET", "https://dashboard.elering.ee/api/nps/price", nil)
	if err != nil {
		return
	}
	q := req.URL.Query()
	trunc := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	q.Add("start", trunc.Format(time.RFC3339))
	q.Add("end", trunc.AddDate(0, 0, 1).Format(time.RFC3339))
	req.URL.RawQuery = q.Encode()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = writeDates(date, bytes)
	if err != nil {
		return
	}
	err = json.Unmarshal(bytes, &prices)
	if err != nil {
		return
	}
	return
}

func writeDates(date time.Time, prices []byte) (err error) {
	f, err := os.Create(pricesFileName(date))
	if err != nil {
		return
	}
	_, err = f.Write(prices)
	return err
}

func pricesFileName(date time.Time) string {
	return fmt.Sprintf("nord_pool_%s.json", date.Format(time.DateOnly))
}

func readPrices(date time.Time) (prices Prices, err error) {
	fileName := pricesFileName(date)
	f, err := os.Open(fileName)
	if err != nil {
		return prices, fmt.Errorf("%s - %w", fileName, errPricesFileDoesNotExist)
	}
	bytes, err := io.ReadAll(f)
	if err != nil {
		return
	}
	err = json.Unmarshal(bytes, &prices)
	return
}
