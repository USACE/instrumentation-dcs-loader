package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	ts "github.com/USACE/instrumentation-api/timeseries"
	"github.com/kelseyhightower/envconfig"

	"github.com/google/uuid"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Config holds parameters parsed from env variables
type Config struct {
	PostURL             string `envconfig:"POST_URL"`
	APIKey              string `envconfig:"API_KEY"`
	AWSS3Endpoint       string `envconfig:"AWS_S3_ENDPOINT"`
	AWSS3Region         string `envconfig:"AWS_S3_REGION"`
	AWSS3DisableSSL     bool   `envconfig:"AWS_S3_DISABLE_SSL"`
	AWSS3ForcePathStyle bool   `envconfig:"AWS_S3_FORCE_PATH_STYLE"`
}

// AWSConfig returns a ready-to-go config to pass to session.New()
// This function helps local testing against minio as an s3 stand-in
// where endpoint must be defined
func (cfg *Config) AWSConfig() *aws.Config {
	awsConfig := aws.NewConfig().WithRegion(cfg.AWSS3Region)

	// Used for "minio" during development
	awsConfig.WithDisableSSL(cfg.AWSS3DisableSSL)
	awsConfig.WithS3ForcePathStyle(cfg.AWSS3ForcePathStyle)
	if cfg.AWSS3Endpoint != "" {
		awsConfig.WithEndpoint(cfg.AWSS3Endpoint)
	}

	return awsConfig
}

// HandlerFunc allows currying HandleRequest
type HandlerFunc func(context.Context, events.S3Event) error

// HandleRequest parses a CSV file hosted on S3 and stores contents instrumentation-api
func HandleRequest(cfg *Config) HandlerFunc {
	return func(ctx context.Context, s3Event events.S3Event) error {

		for _, record := range s3Event.Records {

			sess := session.New(cfg.AWSConfig())
			s3Client := s3.New(sess)

			output, err := s3Client.GetObject(&s3.GetObjectInput{
				Bucket: &record.S3.Bucket.Name,
				Key:    &record.S3.Object.Key,
			})
			if err != nil {
				return err
			}

			// Parse CSV Rows
			reader := csv.NewReader(output.Body)
			rows := make([][]string, 0)
			for {
				row, err := reader.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Fatal(err.Error())
				}
				rows = append(rows, row)
			}

			mcMap := make(map[uuid.UUID]*ts.MeasurementCollection)
			mCount := 0
			for _, r := range rows {
				// 0=timeseries_id, 1=time, 2=value
				// Timeseries ID
				tsid, err := uuid.Parse(r[0])
				if err != nil {
					log.Fatal(err.Error())
				}
				// Time
				t, err := time.Parse(time.RFC3339, r[1])
				if err != nil {
					log.Fatal(err.Error())
				}
				// Value
				v, err := strconv.ParseFloat(r[2], 32)
				if err != nil {
					log.Fatal(err.Error())
				}

				// Create New TimeseriesID: MeasurementCollection Entry in Map (As Necessary)
				if _, ok := mcMap[tsid]; !ok {
					mcMap[tsid] = &ts.MeasurementCollection{
						TimeseriesID: tsid,
						Items:        make([]ts.Measurement, 0),
					}
				}
				// Add Measurement to Measurement Collection
				mcMap[tsid].Items = append(mcMap[tsid].Items, ts.Measurement{TimeseriesID: tsid, Time: t, Value: float32(v)})
				mCount++
			}

			cc := make([]ts.MeasurementCollection, len(mcMap))
			idx := 0
			for _, v := range mcMap {
				cc[idx] = *v
				idx++
			}

			// POST TO INSTRUMENTATION API
			CLIENT := &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return nil
				},
			}
			// Body of Request
			requestBodyBytes, err := json.Marshal(cc)
			if err != nil {
				log.Fatal(err.Error())
			}
			requestBody := bytes.NewReader(requestBodyBytes)

			// Build Request
			r, err := http.NewRequest("POST", fmt.Sprintf("%s?key=%s", cfg.PostURL, cfg.APIKey), requestBody)
			if err != nil {
				log.Fatal(err.Error())
			}

			// Add Headers
			r.Header.Add("Content-Type", "application/json")

			// START THE TIMER
			startPostTime := time.Now()

			// POST
			resp, err := CLIENT.Do(r)
			if err != nil {
				log.Fatalf("\n\t*** Error; %s\n", err.Error())
			}
			if resp.StatusCode == 201 {
				log.Printf(
					"\n\tSUCCESS; POST %d measurements across %d timeseries in %f seconds\n",
					mCount, len(cc), time.Since(startPostTime).Seconds(),
				)
			} else {
				fmt.Printf("\n\t*** Error; Status Code: %d ***\n", resp.StatusCode)
			}
		}
		return nil
	}
}

func main() {

	var cfg Config
	if err := envconfig.Process("loader", &cfg); err != nil {
		log.Fatal(err.Error())
	}
	lambda.Start(HandleRequest(&cfg))
}
