package nordpool

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"io"
	"log"
	"math"
	"net/http"
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
	MaxPrice            float64                `yaml:"max-price"`
	ChargeTillHourDay   int                    `yaml:"charge-till-hour-day"`
	ChargeTillHourNight int                    `yaml:"charge-till-hour-night"`
	Vat                 float64                `yaml:"vat"`
	Timezone            string                 `yaml:"timezone"`
	TransmissionCost    TransmissionCostConfig `yaml:"transmission-cost"`
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

func GetPrice(s3svc *s3.S3, awsS3Bucket string, date time.Time, config NordPoolConfig) (price float64, err error) {
	locationDate, err := locationDate(config, date)
	if err != nil {
		return
	}
	prices, err := getPrices(s3svc, awsS3Bucket, locationDate)
	if err != nil {
		return
	}
	poolPrice, err := findPrice(prices.Data.Lt, locationDate)
	price, err = calculatePrice(locationDate, poolPrice, config)
	return
}

func GetMinPriceTill(s3svc *s3.S3, awsS3Bucket string, date time.Time, config NordPoolConfig) (price float64, err error) {
	locationDate, err := locationDate(config, date)
	if err != nil {
		return
	}
	prices, err := getPrices(s3svc, awsS3Bucket, locationDate)
	if err != nil {
		return
	}
	return findMinPrice(config, prices.Data.Lt, locationDate)
}

func findMinPrice(config NordPoolConfig, prices []Price, locationDate time.Time) (price float64, err error) {
	price = math.MaxFloat64
	chargeTillHour := getChargeTillHour(config, locationDate)
	for locationDate.Hour() != chargeTillHour {
		var poolPrice float64
		poolPrice, err = findPrice(prices, locationDate)
		if err != nil {
			if errors.Is(err, errPriceNotFound) {
				return price, nil
			}
			return
		}
		poolPrice, err = calculatePrice(locationDate, poolPrice, config)
		if poolPrice < price {
			price = poolPrice
		}
		locationDate = locationDate.Add(time.Hour)
	}
	return
}

func getChargeTillHour(config NordPoolConfig, date time.Time) int {
	if date.Hour() > config.ChargeTillHourNight && date.Hour() <= config.ChargeTillHourDay {
		return config.ChargeTillHourDay
	} else {
		return config.ChargeTillHourNight
	}
}

func getPrices(s3svc *s3.S3, awsS3Bucket string, locationDate time.Time) (prices Prices, err error) {
	prices, err = readPrices(s3svc, awsS3Bucket, locationDate)
	if err != nil {
		if !errors.Is(err, errPricesFileDoesNotExist) {
			return
		}
		prices, err = fetchDates(s3svc, awsS3Bucket, locationDate)
		if err != nil {
			return
		}
	}
	return
}

func locationDate(config NordPoolConfig, date time.Time) (locationDate time.Time, err error) {
	location, err := time.LoadLocation(config.Timezone)
	if err != nil {
		return
	}
	locationDate = date.In(location)
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
	log.Printf("Looking for price at %d, date %s", timestamp, date)
	for _, p := range prices {
		if p.Timestamp == timestamp {
			price = p.Price
			return
		}
	}
	return price, fmt.Errorf("%d : %w", timestamp, errPriceNotFound)
}

func fetchDates(s3svc *s3.S3, awsS3Bucket string, date time.Time) (prices Prices, err error) {
	req, err := http.NewRequest("GET", "https://dashboard.elering.ee/api/nps/price", nil)
	if err != nil {
		return
	}
	q := req.URL.Query()
	trunc := time.Date(date.Year(), date.Month(), date.Day(), date.Hour(), 0, 0, 0, date.Location())
	q.Add("start", trunc.Format(time.RFC3339))
	q.Add("end", trunc.AddDate(0, 0, 1).Format(time.RFC3339))
	log.Printf("Fetching prices from %s to %s", q.Get("start"), q.Get("end"))
	req.URL.RawQuery = q.Encode()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	pricesBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = writeDates(s3svc, awsS3Bucket, date, pricesBytes)
	if err != nil {
		return
	}
	err = json.Unmarshal(pricesBytes, &prices)
	if err != nil {
		return
	}
	return
}

func writeDates(s3svc *s3.S3, awsS3Bucket string, date time.Time, prices []byte) (err error) {
	_, err = s3svc.PutObject(&s3.PutObjectInput{
		Body:   bytes.NewReader(prices),
		Bucket: &awsS3Bucket,
		Key:    aws.String(pricesFileName(date)),
	})
	return err
}

func pricesFileName(date time.Time) string {
	return fmt.Sprintf("nord_pool_%s_%d.json", date.Format(time.DateOnly), date.Hour())
}

func readPrices(s3svc *s3.S3, awsS3Bucket string, date time.Time) (prices Prices, err error) {
	fileName := pricesFileName(date)
	log.Printf("Reading prices from %s", fileName)
	input := &s3.GetObjectInput{Bucket: aws.String(awsS3Bucket),
		Key: &fileName,
	}
	output, err := s3svc.GetObject(input)
	if err != nil {
		return prices, fmt.Errorf("%s - %w", fileName, errPricesFileDoesNotExist)
	}
	defer output.Body.Close()
	pricesBytes, err := io.ReadAll(output.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(pricesBytes, &prices)
	return
}
