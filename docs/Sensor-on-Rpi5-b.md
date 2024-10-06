# Running a PING42 Sensor on a Rapsberry Pi 5

Raspberry Pi devices make for pretty good sensors, with modern features such as modern ARM-based CPUs and high timer precision, they are excellent for measuring the conditions on the global internet.

This guide is written for Rapsberry Pi `Raspberry Pi 5 Model B Rev 1.0`, but should cover most similar small computer boards. Do not be shy to use Google to find out whcih board you have at hand if you are not certain.

## Connectivity Considerations

Ethernet is recommended. The latency variability of WiFi networks make them unsitable for the type of measurments we need. The accuracy of the telemetry of how the whole internet network behaves will be diminished if the sensor itself has unstable latency to your router.

Please consider connecting the sensor direclty to your internet router via a network cable and using remote tools to connect to it, such as SSH.

## Installing Raspbian OS

Some Rapsberry Pi kits come with [Raspbian](https://www.raspberrypi.com/software/) OS pre-installed, others derivative models of the official Pi have modified versions to support the extended functionality of some boards.

Please refer to the official documentation of your board in case is slightly different than the stock Pi 5 Model B.

## SSH and Remote Management

It is generally a good idea to allow yourself to connect to the RPi board over your home network. This can be achieved by setting up an OpenSSH Server on the device. Please refer to the [official documentation](https://www.raspberrypi.com/documentation/computers/remote-access.html) on how to configure remote access.

We also recommend disabling the graphical environment by adjusting the autologin settings with the `raspi-config` tool.

## Installing Docker

The PING42 sensor is distributed as a container, and requires Docker Engine or other compatible container runtimes if you prefer. To install Docker, please refer to their official guide for Debian-based distributions [here](https://docs.docker.com/engine/install/debian/).

At the end, you should be able to pull the sensor image locally like so:

```bash
docker pull ghcr.io/ping-42/sensor
```

If this fails for some reason, please ensure that your Docker is set up correctly.

## Clocks, Timezone and Accuracy

Keeping the device time syncrinized is extremely important for the accuracy of telemetry data. Luckily, the kind folks at Rapsbian enable Network Time Protocol (NTP) synchronization by default. 

In order to view the current timezone of the device, the `timedatectl` command can be used:

```bash
$ timedatectl
               Local time: Sun 2024-10-06 17:57:57 JST
           Universal time: Sun 2024-10-06 08:57:57 UTC
                 RTC time: Sun 2024-10-06 08:57:57
                Time zone: Asia/Tokyo (JST, +0900)
System clock synchronized: yes
              NTP service: active
          RTC in local TZ: no
```

## Cooling and Temperature

Depending on your board and kit, various cooling options might be available. We generally prefer passive cooling options (ie without fans) because it makes the board much quieter and greatly extends its lifetime. Cooling of some sort is generally required for Raspberry Pi boards, otherwise the device will start misbehaving as it is overheating.

## Deploying the Sensor

Please refer to the latest sensor documentation in its repository at github.com/ping-42/sensor.
