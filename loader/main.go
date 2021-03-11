package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	ts "github.com/USACE/instrumentation-api/timeseries"
	"github.com/kelseyhightower/envconfig"

	"github.com/google/uuid"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// Config holds parameters parsed from env variables.
// Note awsSQSQueueURL private variable. Public method is AWSSQSQueueURL()
type Config struct {
	PostURL             string `envconfig:"POST_URL"`
	APIKey              string `envconfig:"API_KEY"`
	AWSS3Region         string `envconfig:"AWS_S3_REGION"`
	AWSS3Endpoint       string `envconfig:"AWS_S3_ENDPOINT"`
	AWSS3DisableSSL     bool   `envconfig:"AWS_S3_DISABLE_SSL"`
	AWSS3ForcePathStyle bool   `envconfig:"AWS_S3_FORCE_PATH_STYLE"`
	AWSSQSRegion        string `envconfig:"AWS_SQS_REGION"`
	AWSSQSEndpoint      string `envconfig:"AWS_SQS_ENDPOINT"`
	awsSQSQueueURL      string `envconfig:"AWS_SQS_QUEUE_URL"`
	AWSSQSQueueName     string `envconfig:"AWS_SQS_QUEUE_NAME"`
}

// AWSS3Config returns a ready-to-go config to pass to session.New() for S3
// This function helps local testing against minio as an s3 stand-in
// where endpoint must be defined
func (cfg *Config) AWSS3Config() *aws.Config {
	awsConfig := aws.NewConfig()

	// Used for "minio" during development
	awsConfig.WithDisableSSL(cfg.AWSS3DisableSSL)
	awsConfig.WithS3ForcePathStyle(cfg.AWSS3ForcePathStyle)

	if cfg.AWSS3Region != "" {
		awsConfig.WithRegion(cfg.AWSS3Region)
	}
	if cfg.AWSS3Endpoint != "" {
		awsConfig.WithEndpoint(cfg.AWSS3Endpoint)
	}

	return awsConfig
}

// AWSSQSConfig returns a ready-to-go config for session.New() for SQS Actions.
// Supports local testing using SQS stand-in elasticmq
func (cfg *Config) AWSSQSConfig() *aws.Config {
	awsConfig := aws.NewConfig()
	if cfg.AWSSQSRegion != "" {
		awsConfig.WithRegion(cfg.AWSSQSRegion)
	}
	if cfg.AWSSQSEndpoint != "" {
		awsConfig.WithEndpoint(cfg.AWSSQSEndpoint)
	}
	if cfg.AWSSQSRegion != "" {
		awsConfig.WithRegion(cfg.AWSSQSRegion)
	}
	return awsConfig
}

// AWSSQSQueueURL returns the QueueUrl for interacting with SQS
func (cfg *Config) AWSSQSQueueURL(s *sqs.SQS) (string, error) {
	// If environment variable AWS_SQS_QUEUE_URL is specified,
	// use provided queue URL without question
	if cfg.awsSQSQueueURL != "" {
		return cfg.awsSQSQueueURL, nil
	}
	// Lookup Queue URL from AWS_SQS_QUEUE_NAME
	urlResult, err := s.GetQueueUrl(&sqs.GetQueueUrlInput{QueueName: &cfg.AWSSQSQueueName})
	if err != nil {
		return "", err
	}
	return *urlResult.QueueUrl, nil
}

// HandlerFunc allows currying HandleRequest
type HandlerFunc func(context.Context, events.S3Event) error

// HandleRequest parses a CSV file hosted on S3 and stores contents instrumentation-api
func HandleRequest(cfg *Config) HandlerFunc {
	return func(ctx context.Context, s3Event events.S3Event) error {

		for _, record := range s3Event.Records {

			sess := session.New(cfg.AWSS3Config())
			s3Client := s3.New(sess)

			bucket, key := &record.S3.Bucket.Name, &record.S3.Object.Key
			fmt.Printf("Processing File; bucket: %s; key: %s\n", *bucket, *key)

			output, err := s3Client.GetObject(&s3.GetObjectInput{Bucket: bucket, Key: key})
			if err != nil {
				log.Println(err.Error())
				return err
			}

			// Parse CSV Rows
			defer output.Body.Close()
			reader := csv.NewReader(output.Body)

			rows := make([][]string, 0)
			for {
				row, err := reader.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
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
					return err
				}
				// Time
				t, err := time.Parse(time.RFC3339, r[1])
				if err != nil {
					return err
				}
				// Value
				v, err := strconv.ParseFloat(r[2], 32)
				if err != nil {
					return err
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
				return err
			}
			requestBody := bytes.NewReader(requestBodyBytes)

			// Build Request
			r, err := http.NewRequest("POST", fmt.Sprintf("%s?key=%s", cfg.PostURL, cfg.APIKey), requestBody)
			if err != nil {
				return err
			}

			// Add Headers
			r.Header.Add("Content-Type", "application/json")

			// START THE TIMER
			startPostTime := time.Now()

			// POST
			resp, err := CLIENT.Do(r)
			if err != nil {
				log.Printf("\n\t*** Error; %s\n", err.Error())
				return err
			}

			if resp.StatusCode == 201 {
				fmt.Printf(
					"\n\tSUCCESS; POST %d measurements across %d timeseries in %f seconds\n",
					mCount, len(cc), time.Since(startPostTime).Seconds(),
				)
			} else {
				fmt.Printf("\n\t*** Error; Status Code: %d ***\n", resp.StatusCode)
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					fmt.Println("Error reading response body")
					return err
				}
				log.Printf("%s\n", body)
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

	// Handler
	handler := HandleRequest(&cfg)

	// SQS Session
	sessSQS := session.New(cfg.AWSSQSConfig())
	svcSQS := sqs.New(sessSQS)

	queueURL, err := cfg.AWSSQSQueueURL(svcSQS)
	if err != nil {
		log.Fatal(err.Error())
	}
	if queueURL == "" {
		log.Fatal("Could not find queue url")
	}

	// Single memory locations to be reused by all for loop iterations
	// sns
	var SNSEvt events.SNSEntity
	pSNSEvt := &SNSEvt
	// s3
	var S3Evt events.S3Event
	pS3Evt := &S3Evt

	for {
		fmt.Println("Calling Receive Messages...")
		output, err := svcSQS.ReceiveMessage(&sqs.ReceiveMessageInput{
			AttributeNames: []*string{
				aws.String(sqs.MessageSystemAttributeNameSentTimestamp),
			},
			MessageAttributeNames: []*string{
				aws.String(sqs.QueueAttributeNameAll),
			},
			QueueUrl:            &queueURL,
			MaxNumberOfMessages: aws.Int64(1),
			VisibilityTimeout:   aws.Int64(30),
			WaitTimeSeconds:     aws.Int64(20),
		})
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
			continue
		}

		fmt.Printf("Received %d messages\n", len(output.Messages))
		for _, m := range output.Messages {
			fmt.Printf("Working on Message: %s\n", *m.MessageId)

			// Unmarshal entire message body into SNS Entity
			if err := json.Unmarshal([]byte(*m.Body), pSNSEvt); err != nil {
				fmt.Printf("Error: %s\n", err.Error())
				continue
			}

			// Unmarshal Message to SNSEvent
			if err := json.Unmarshal([]byte(pSNSEvt.Message), pS3Evt); err != nil {
				fmt.Printf("Error: %s\n", err.Error())
				continue
			}

			// Handle Message
			if err := handler(context.Background(), *pS3Evt); err != nil {
				fmt.Printf("Error: %s\n", err.Error())
				continue
			}

			svcSQS.DeleteMessage(&sqs.DeleteMessageInput{
				QueueUrl:      &queueURL,
				ReceiptHandle: m.MessageId,
			})
		}
	}
}
