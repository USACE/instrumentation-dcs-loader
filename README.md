# instrumentation-dcs-loader

AWS Lambda Function code to parse CSV Files of timeseries measurements on AWS S3 and post contents to [instrumentation-api](https://github.com/USACE/instrumentation-api)

### Environment Variables


   | Variable                | Example Value                                          |
   | ----------------------- | ------------------------------------------------------ |
   | LOADER_POST_URL         | http://instrumentation-api.dev/timeseries_measurements |
   | LOADER_API_KEY          | appkey                                                 |
   | AWS_S3_ENDPOINT         | http://localhost:9000                                  |
   | AWS_S3_REGION           | us-east-1                                              |
   | AWS_S3_DISABLE_SSL      | False                                                  |
   | AWS_S3_FORCE_PATH_STYLE | True                                                   |


### Example Input File

```
869465fc-dc1e-445e-81f4-9979b5fadda9,2021-03-01T15:30:00Z,27.6800
869465fc-dc1e-445e-81f4-9979b5fadda9,2021-03-01T15:00:00Z,27.6200
869465fc-dc1e-445e-81f4-9979b5fadda9,2021-03-01T14:30:00Z,27.5500
869465fc-dc1e-445e-81f4-9979b5fadda9,2021-03-01T14:00:00Z,27.4400
```
