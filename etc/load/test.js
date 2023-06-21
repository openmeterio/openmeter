import http from 'k6/http';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';
import { check } from 'k6';
import { sleep } from 'k6';

export const options = {
  vus: 1,
  iterations: 100,
};

export function setup() {
  return { subject: uuidv4() };
}

export default function (data) {
  const payload = JSON.stringify({
    specversion: "1.0",
    type: "api-calls",
    id: uuidv4(),
    source: "service-0",
    subject: data.subject,
    data: {
      duration_ms: "1",
      method: "GET",
      path: "/hello"
    }
  });

  const headers = { 'Content-Type': 'application/cloudevents+json' };

  const res = http.post('http://localhost:8888/api/v1alpha1/events', payload, { headers });

  check(res, {
    'is status 200': (r) => r.status === 200,
  });
}

export function teardown(data) {
  sleep(10);

  const res = http.get('http://localhost:8888/api/v1alpha1/meters/m1/values');

  const values = JSON.parse(res.body);

  values.data.forEach(element => {
    if (element.subject == data.subject) {
      console.log("Element:", element.value);
    }
  });
}
