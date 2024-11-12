[![Golang Pipeline](https://github.com/ping-42/sensor/actions/workflows/go-pipeline.yml/badge.svg)](https://github.com/ping-42/sensor/actions/workflows/go-pipeline.yml)
# PING42 Sensor

The sensor is the application client of the PING42 community. Our members run it on their systems and contribute internet telemetry to our project. The sensor connects to a centralized API and is asigned work to do, in the form of periodically conducting small tests connections to various places on the internet.

Each test is measured as accurately as possible for its time, latency and performance and the measurement is then submitted to community operated PING42 infrastructure which warehouses and aggregates data.

Community participants who wish to provide a small amount of their internet connection run the sensor in exchange for being compensated by various users around the world to test their applications behavior in the real world.

Most uptime monitoring tools out there test from few, centralized locations, which do not reflect the diverse internet ecosystem around the globe. For more details on the mission of the PING42 DAO and its community, please visit https://ping42.net and join our Discord server.

> Please note that this guide assums modereate experience with development tools for Linux such as the ability to run Docker containers.

## Running a Sensor

The first step to opearting a sensor is to generate a Sensor Token. The token is unique to each sensor and is used by the protocol to identify the various tasks being asigned to each sensor. It is recommended to run a seperate sensor for each internet connection you want to monitor. Please refer to the more advanced deployment recommendations further down this page.

Armed with a Sensor Token, it is now time to start the sensor with Docker:

```bash
export PING42_SENSOR_TOKEN="XXXXX"
docker run -e PING42_SENSOR_TOKEN="${PING42_SENSOR_TOKEN}" -d --name ping42-sensor --restart=always ghcr.io/ping-42/sensor:latest
```

The `-d` flag ensures the sensor runs in the background. Its logs are available via `docker logs -f ping42-sensor` in the same terminal.

This should start sending telemetry to the network immediately and you should see the sensor as being online.

## Keeping the Sensor Updated

In order to keep a container up to date on a fairly standard local Docker installation, we commend running something like Watchtower.

The following will check for sensor updates every 24 hours and is highly recommended:

```bash
docker run -d \
  --name watchtower \
  --restart=always \
  -v /var/run/docker.sock:/var/run/docker.sock \
  containrrr/watchtower ping42-sensor
```

## Connectivity Considerations

We recommend that the sensor is deployed on a system which is connected to the internet via a wired connection. Home WiFi networks add latency to the network which affects the accuracy of the telemetry data. Whenever possible, please consider using Ethernet cables directly to your ISP router.

Home network routing devices, usually supplied by service providers - called [CPE](https://en.wikipedia.org/wiki/Customer-premises_equipment) for short) - provide some functionality which could affect network latency and experince. ISPs are known to enable things like hijacking DNS responses to missing records and instead supplying their own replacement websites. Please take care to disable such functionality so that the sensor can get a more realistic consideration. 

Once your sensor is up and running, you can perform a quick speedtest to ensure the device is connected correctly and stable:

```
$ docker run -it gists/speedtest-cli speedtest --accept-license

   Speedtest by Ookla

      Server: IPA CyberLab 400G - Tokyo (id: 48463)
         ISP: So-net
Idle Latency:     2.27 ms   (jitter: 0.22ms, low: 2.09ms, high: 2.63ms)
    Download:   950.83 Mbps (data used: 428.1 MB)
                  9.15 ms   (jitter: 0.50ms, low: 2.26ms, high: 10.64ms)
      Upload:   760.86 Mbps (data used: 343.6 MB)
                 10.28 ms   (jitter: 0.70ms, low: 2.16ms, high: 11.66ms)
 Packet Loss:     0.0%
  Result URL: https://www.speedtest.net/result/c/fc34e3e7-82e1-4399-b660-974e402a3b3d
```


Note that in reality the sensor is designed to utilize barely a few kilobits of your internet connection. It is a good idea to periodically speedtest your internet link to ensure that connectivity works fine.

## Alternative Installation

In order to download a binary sensor build, its SBOM and various release artifacts, pelase head over to the [Releases](https://github.com/ping-42/sensor/releases) page.
