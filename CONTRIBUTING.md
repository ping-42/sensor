# Sensor Development

The sensor aims to be an advanced measurment instrument for the modern internet. We are very excited to receive your future contributions and are looking forward to hacking together.

## Testing & Development

Development is best done via a Github Codespace or a local devcontainer - head over to the [42lib](https://github.com/ping-42/42lib) page for more information on contributing code. We really appreciate your interest!

Run a local binary:

```bash
export PING42_TELEMETRY_SERVER="ws://localhost:8080"
go run .
```

## Shipping a new Sensor version

To ship a new version, simply create a tag and Goreleaser will take care of the release process with an action with a new image being generated in [Packages](https://github.com/ping-42/sensor/pkgs/container/sensor).
