job "influxdb" {
  datacenters = ["dc1"]
  type = "service"

  group "influxdb" {
    count = 1

    task "influxdb" {
      driver = "docker"

      config {
        image = "influxdb:latest"
        network_mode = "host"
      }

    }
  }
}
