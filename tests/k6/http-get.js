import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  vus: 1000, // 10k broke the vm
  duration: '30s',
};

export default function () {
  const url = "http://influx.emilsallem.com/api/v2/write?org=34187da0646d3fb8&bucket=test&precision=ms";

  // get current timestamp in ms
  let timestamp = Date.now() ;

  const payload = `
    airSensors,sensor_id="K6_${__VU}" temperature=73.97038159354763,humidity=35.23103248356096,co=0.48445310567793615 ${timestamp}
    airSensors,sensor_id="K6_${__VU}" temperature=75.30007505999716,humidity=35.651929918691714,co=0.5141876544505826 ${timestamp}
    `;

  const params =  {
    headers: {
      "Authorization": "Token ",
      "Content-Type": "text/plain; charset=utf-8",
      "Accept": "application/json",
    },
  };

  let res = http.post(url, payload, params);

  // Check if the request was successful
  check(res, {
    'status was 200 or 204': (r) => r.status === 200 || r.status === 204,
  });

  sleep(1);
}
