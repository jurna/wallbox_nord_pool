package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
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
	//lambda.Start(run)
	err := run()
	if err != nil {
		log.Fatalf("Fatal error: %v", err)
	}
}

func run() error {
	awsRegion := os.Getenv("AWS_REGION")
	awsS3Bucket := os.Getenv("AWS_S3_BUCKET")
	sess, err := session.NewSession(&aws.Config{Region: aws.String(awsRegion)})
	if err != nil {
		return err
	}
	svc := s3.New(sess)
	err, config := readConfig(svc, awsS3Bucket)
	if err != nil {
		return err
	}
	wb, err := wallbox.NewWallbox(config.Wallbox, svc, awsS3Bucket)
	if err != nil {
		return err
	}
	price, err := nordpool.GetPrice(svc, awsS3Bucket, time.Now(), config.NordPool)
	if err != nil {
		return err
	}
	desiredPrice, err := desiredPrice(svc, awsS3Bucket, config)
	if err != nil {
		return err
	}
	status, err := wb.GetStatus()
	if err != nil {
		return err
	}
	var flowState = flow.NewFlowsState(price, desiredPrice, status)
	log.Printf("Flow for state %s, price %f, desiredPrice %f", flowState, price, desiredPrice)
	err = flow.DoFlow(flowState)(wb, price)
	if err != nil {
		return err
	}
	return nil
}

func desiredPrice(svc *s3.S3, awsS3Bucket string, config Config) (desiredPrice float64, err error) {
	minPrice, err := nordpool.GetMinPriceTill(svc, awsS3Bucket, time.Now(), config.NordPool)
	if err != nil {
		return
	}
	if minPrice > config.NordPool.MaxPrice {
		desiredPrice = config.NordPool.MaxPrice
	} else {
		desiredPrice = minPrice
	}
	return
}

func readConfig(svc *s3.S3, awsS3Bucket string) (err error, config Config) {
	input := &s3.GetObjectInput{Bucket: aws.String(awsS3Bucket),
		Key: aws.String("config.yaml"),
	}
	output, err := svc.GetObject(input)
	if err != nil {
		return
	}
	defer output.Body.Close()
	configBytes, err := io.ReadAll(output.Body)
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
