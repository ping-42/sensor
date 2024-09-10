[![Golang Pipeline](https://github.com/ping-42/sensor/actions/workflows/go-pipeline.yml/badge.svg)](https://github.com/ping-42/sensor/actions/workflows/go-pipeline.yml)
# PING42 Sensor

The PING42 Sensor is the application client community members run on their systems connection to start contributing internet telemetry. The sensor connects to a centralized API and is asigned work to do, in the form of periodically conducting small tests connections to various places on the internet. Each test is measured as accurately as possible for its time, latency and performance and the measurement is then submitted to community operated PING42 infrastructure which warehouses and aggregates data. 

Community participants who wish to provide a small amount of their internet connection run the sensor package in exchange for being compensated by various users around the world to test their applications behavior in real the real world. Most uptime monitoring tools out there test from few, centralized locations, which do not reflect the diverse internet connections around the globe. For more details on the mission of the PING42 DAO and its community, please visit https://ping42.net and join our Discord server.

> Please note that this guide assums modereate development experience, such as the ability to install Docker.

## Running a Sensor

The first step to opearting a sensor is to generate a Sensor Token. The token is unique to each sensor and is used by the protocol to identify the various tasks being asigned to each sensor. It is recommended to run a seperate sensor for each internet connection you want to monitor. Please refer to the more advanced deployment recommendations further down this page.

Armed with a Sensor Token, it is now time to start the sensor with Docker:

```bash
export PING42_SENSOR_TOKEN="XXXXX"
docker run -e PING42_SENSOR_TOKEN="${PING42_SENSOR_TOKEN}" -d --name ping42-sensor --restart=always ghcr.io/ping-42/sensor:latest
```

The `-d` flag ensures the sensor runs in the background. Its logs are available via `docker logs -f ping42-sensor` in the same terminal.

This should start sending telemetry to the network immediately and you should see the sensor as being online.

## Alternative Installation

In order to download a binary sensor build, its SBOM and various release artifacts, pelase head over to the [Releases](https://github.com/ping-42/sensor/releases) page.
