# instrumentation-dcs-loader

SQS-Worker to parse CSV Files of timeseries measurements on AWS S3 and post contents to [instrumentation-api](https://github.com/USACE/instrumentation-api). 

Works with ElasticMQ for local testing with a SQS-compatible interface. Variables noted "Used for local testing" typically do not need to be provided when deployed, for example to AWS. They can be omitted completely or set to "" if not required.

### Environment Variables

| Variable                       | Example Value                                                            | Notes                  |
| ------------------------------ | ------------------------------------------------------------------------ | ---------------------- |
| AWS_ACCESS_KEY_ID              | AKIAIOSFODNN7EXAMPLE                                                     | Used for local testing |
| AWS_SECRET_ACCESS_KEY          | wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY                                 | Used for local testing |
| AWS_DEFAULT_REGION             | us-east-1                                                                | Used for local testing |
| ------------------------------ | ------------------------------------------------------------------------ | ---------------------- |
| LOADER_POST_URL                | http://instrumentation-api_api_1/instrumentation/timeseries_measurements |                        |
| LOADER_API_KEY                 | appkey                                                                   |                        |
| ------------------------------ | ------------------------------------------------------------------------ | ---------------------- |
| LOADER_AWS_S3_ENDPOINT         | http://minio:9000                                                        | Used for local testing |
| LOADER_AWS_S3_REGION           | us-east-1                                                                |                        |
| LOADER_AWS_S3_DISABLE_SSL      | False                                                                    |                        |
| LOADER_AWS_S3_FORCE_PATH_STYLE | True                                                                     |                        |
| ------------------------------ | ------------------------------------------------------------------------ | ---------------------- |
| LOADER_AWS_SQS_QUEUE_NAME      | instrumentation-dcs-goes                                                 |                        |
| LOADER_AWS_SQS_ENDPOINT        | http://elasticmq:9324                                                    | Used for local testing |
| LOADER_AWS_SQS_QUEUE_URL       | http://elasticmq:9324/queue/instrumentation-dcs-goes                     | Used for local testing |
| LOADER_AWS_SQS_REGION          | elasticmq                                                                | Used for local testing |

### Example Input File

```
869465fc-dc1e-445e-81f4-9979b5fadda9,2021-03-01T15:30:00Z,27.6800
869465fc-dc1e-445e-81f4-9979b5fadda9,2021-03-01T15:00:00Z,27.6200
869465fc-dc1e-445e-81f4-9979b5fadda9,2021-03-01T14:30:00Z,27.5500
869465fc-dc1e-445e-81f4-9979b5fadda9,2021-03-01T14:00:00Z,27.4400
```
