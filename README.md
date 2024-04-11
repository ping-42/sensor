# PING42 Sensor

Follow these instructions in order to start contributing telemetry to the PING42 protocol, help study the internet, ensure uptime for a whole universe of applications and receive rewards for it.

## Running a Sensor

After obtaining a Sensor token, starting a sensor on any system with a working Docker installation is very simple:
```bash
docker run --rm -ti -e PING42_SENSOR_TOKEN="XXXXX" ghcr.io/ping-42/sensor:latest
```

This should start sending telemetry to the network immediately and you should see the sensor as being online.

## Alternative Installation

In order to download a binary sensor build, its SBOM and various release artifacts, pelase head over to the [Releases](https://github.com/ping-42/sensor/releases) page.

## Testing & Development

Development is best done via a Github Codespace - head over to the [42lib](https://github.com/ping-42/42lib) page for more information on contributing code. We really appreciate your interest!

```bash
go run .
```
