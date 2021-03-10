package main

import (
	"encoding/json"
	"log"

	"github.com/USACE/go-simple-asyncer/asyncer"
	"github.com/aws/aws-lambda-go/events"
)

func main() {

	// MSG_COUNT := 10

	a, err := asyncer.NewAsyncer(asyncer.Config{Engine: "AWSSQS", Target: "local/http://localhost:9324/queue/instrumentation-dcs-goes"})
	if err != nil {
		log.Panic(err.Error())
	}

	// S3 Record
	r := events.S3EventRecord{
		S3: events.S3Entity{
			Bucket: events.S3Bucket{Name: "corpsmap-data-incoming"},
			Object: events.S3Object{Key: "test/test-file.csv"},
		},
	}

	eventStr, err := json.Marshal(events.S3Event{Records: []events.S3EventRecord{r}})
	if err != nil {
		log.Panic(err.Error())
	}

	snsClone, err := json.Marshal(events.SNSEntity{Message: string(eventStr)})
	if err != nil {
		log.Panic(err.Error())
	}

	if err := a.CallAsync(snsClone); err != nil {
		log.Panic(err.Error())
	}

}
