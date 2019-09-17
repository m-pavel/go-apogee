all: mqtt influx

deps:
	go get -v -d ./...

mqtt: deps
	go build -o apogee-influx ./cmd/exporter

influx: deps
	go build -o apogee-mqtt ./cmd/mqtt

clean:
	rm -f apogee-mqtt
	rm -f apogee-influx


