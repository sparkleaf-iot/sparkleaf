void setup() {
  Serial.begin(9600);
  pinMode(A0, INPUT_PULLUP); 
  pinMode(A2, INPUT_PULLUP); 

}

void loop() {
  int sensorValue = analogRead(A0);
  int sensorValue2 = analogRead(A2);
  float voltage = sensorValue * (5.0 / 1023.0);
  float voltage2 = sensorValue2 * (5.0 / 1023.0);

  delay(500);
  // Printing sensor values to be read by esp
  Serial.print(sensorValue);
  Serial.print(",");
  Serial.println(sensorValue2);

}
