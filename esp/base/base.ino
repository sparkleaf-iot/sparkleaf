#if defined(ESP32)
  #include <WiFiMulti.h>
  WiFiMulti wifiMulti;
  #define DEVICE "ESP32"
  #elif defined(ESP8266)
  #include <ESP8266WiFiMulti.h>
  ESP8266WiFiMulti wifiMulti;
  #define DEVICE "ESP8266"
  #endif
  
  #include <InfluxDbClient.h>
  #include <InfluxDbCloud.h>
  
  // WiFi AP SSID
  #define WIFI_SSID ""
  // WiFi password
  #define WIFI_PASSWORD ""
  
  #define INFLUXDB_URL "http://influx.emilsallem.com"
  #define INFLUXDB_TOKEN ""
  #define INFLUXDB_ORG ""
  #define INFLUXDB_BUCKET "test"
  
  // Time zone info
  #define TZ_INFO "UTC2"
  const int input1 = 34;
  const int input2 = 35;
  int sensingValue2 = 0;
  int sensingValue1 = 0;
  String var1;
  String var2;
  #define RXp2 16
  #define TXp2 17
  // Declare InfluxDB client instance with preconfigured InfluxCloud certificate
  InfluxDBClient client(INFLUXDB_URL, INFLUXDB_ORG, INFLUXDB_BUCKET, INFLUXDB_TOKEN, InfluxDbCloud2CACert);
  
  // Declare Data point
  Point sensor("wifi_status");
  void setup() {
    Serial.begin(115200);
    Serial2.setTimeout(100);
    Serial2.begin(9600, SERIAL_8N1, RXp2, TXp2);
    pinMode(34, INPUT_PULLUP);
    pinMode(35, INPUT_PULLUP);

    // Setup wifi
    WiFi.mode(WIFI_STA);
    wifiMulti.addAP(WIFI_SSID, WIFI_PASSWORD);
  
    Serial.print("Connecting to wifi");
    while (wifiMulti.run() != WL_CONNECTED) {
      Serial.print(".");
      delay(100);
    }
    Serial.println();
  
    // Accurate time is necessary for certificate validation and writing in batches
    // Syncing progress and the time will be printed to Serial.
    timeSync(TZ_INFO, "pool.ntp.org", "time.nis.gov");
  
  
    // Check server connection
    if (client.validateConnection()) {
      Serial.print("Connected to InfluxDB: ");
      Serial.println(client.getServerUrl());
    } else {
      Serial.print("InfluxDB connection failed: ");
      Serial.println(client.getLastErrorMessage());
    }

    // Add tags to the data point
    sensor.addTag("device", DEVICE);
    sensor.addTag("SSID", WiFi.SSID());

  }
  void loop() {
     // Clear fields for reusing the point. Tags will remain the same as set above.
    sensor.clearFields();
    var1 = Serial2.readStringUntil(','); // writes in the string all the inputs till a comma
    Serial.read(); 
    var2 = Serial2.readStringUntil('\n'); // writes in the string all the inputs till the end of line character


    // Store measured value into point
    // Report RSSI of currently connected network
    sensor.addField("rssi", WiFi.RSSI());
    sensor.addField("bitValue_1", var1.toInt());
    sensor.addField("bitValue_2", var2.toInt());


    // Print what are we exactly writing
    Serial.print("Writing: ");
    Serial.println(sensor.toLineProtocol());

  
    // Check WiFi connection and reconnect if needed
    if (wifiMulti.run() != WL_CONNECTED) {
      Serial.println("Wifi connection lost");
    }
  
    //Write point
    if (!client.writePoint(sensor)) {
      Serial.print("InfluxDB write failed: ");
      Serial.println(client.getLastErrorMessage());
    }
  
    delay(500);


  }
