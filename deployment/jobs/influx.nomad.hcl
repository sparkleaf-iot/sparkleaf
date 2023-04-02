job "influxdb" {
  datacenters = ["dc1"]

  group "influxdb" {

    volume "influxdb" {
      type      = "host"
      source    = "influxdb"
      read_only = false
    }

    network {
      port  "http"{
         static = 8086
      }
    }
    task "influxdb" {
      driver = "docker"
      volume_mount {
        volume      = "influxdb"
        destination = "/var/lib/influxdb2"
        read_only   = false
      }

      config {
        image = "influxdb:2.6-alpine"
        ports = ["http"]
      }

      env {
        INFLUXDB_DB = "mydb"
        INFLUXDB_ADMIN_USER = "admin"
        INFLUXDB_ADMIN_PASSWORD = "password"
      }

      service {
        name = "influxdb"
        port = "http"
        provider = "nomad"
        check {
          type     = "tcp"
          interval = "10s"
          timeout  = "2s"
        }
      }
    }
  }
}
