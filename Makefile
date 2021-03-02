packagename = instrumentation-dcs-loader.zip

.PHONY: build clean

build:
	env GOOS=linux go build -ldflags="-s -w" -o bin/instrumentation-dcs-loader loader/main.go

clean:
	rm -rf ./bin $(packagename)

package: clean build
	zip -r $(packagename) bin

deploy-dev: package
	aws s3 cp $(packagename) s3://corpsmap-lambda-zips/$(packagename)

deploy-test: package
	aws s3 cp $(packagename) s3://rsgis-lambda-zips/$(packagename)

test: clean build
	./test.sh
