package main

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"os"
	"time"
	"wallbox_nord_pool/internal/flow"
	"wallbox_nord_pool/internal/nordpool"
	"wallbox_nord_pool/internal/wallbox"
)

func main() {
	err, config := readConfig()
	if err != nil {
		log.Fatalf("Fatal error: %s", err)
	}
	wb, err := wallbox.NewWallbox(config.Wallbox)
	if err != nil {
		log.Fatalf("Fatal error: %s", err)
	}
	price, err := nordpool.GetPrice(time.Now(), config.NordPool)
	if err != nil {
		log.Fatalf("Fatal error: %s", err)
	}
	status, err := wb.GetStatus()
	if err != nil {
		log.Fatalf("Fatal error: %s", err)
	}
	var flowState = flow.NewFlowsState(price, config.NordPool.MaxPrice, status)
	log.Printf("Flow for state %s, price %f", flowState, price)
	err = flow.DoFlow(flowState)(wb, price)
	if err != nil {
		log.Fatalf("Fatal error: %s", err)
	}

}

func readConfig() (err error, config Config) {
	f, err := os.Open("config.yaml")
	if err != nil {
		return fmt.Errorf("config file not found: %w", err), config
	}
	configBytes, err := io.ReadAll(f)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(configBytes, &config)
	if err != nil {
		return
	}
	return
}

type Config struct {
	NordPool nordpool.NordPoolConfig `yaml:"nord-pool"`
	Wallbox  wallbox.Config          `yaml:"wallbox"`
}
