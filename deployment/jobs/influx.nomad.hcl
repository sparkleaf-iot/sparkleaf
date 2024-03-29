job "influxdb" {
  datacenters = ["dc1"]

  group "influxdb" {

    volume "influxdb" {
      type      = "csi"
      source    = "influx_volume"
      read_only = false
      access_mode = "single-node-writer"
      attachment_mode = "file-system"
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
        image = "influxdb:2.7-alpine"
        ports = ["http"]
      }

      env {
        INFLUXDB_DB = "maindb"
        INFLUXDB_ADMIN_USER = "admin"
        INFLUXDB_ADMIN_PASSWORD = "password"
      }

      service {
        name = "influxdb"
        port = "http"
        provider = "consul"
        tags = [
        "traefik.enable=true",
        "traefik.http.routers.influxdb.rule=Host(`influx.emilsallem.com`)",
        "traefik.http.routers.influxdb.entrypoints=web",
      ]
        
      }
    }
  }
}
