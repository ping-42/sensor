# Ping42 Sensor

This repository contains the code that belongs to the Ping 42 Sensor. Its code builds
into a docker image that can then be deployed in various locations.

## Testing
It is currently set up to test locally on localhost:8080.

```bash
go mod vendor
docker build . -t ping-42/sensor
```
- run the container

```bash
docker run --rm -p 8080:8080 --name ping42-sensor ping-42/sensor
```