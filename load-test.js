import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

export let errorRate = new Rate('errors');

export let options = {
  stages: [
    { duration: '2m', target: 100 }, // Ramp up to 100 users
    { duration: '5m', target: 100 }, // Stay at 100 users
    { duration: '2m', target: 200 }, // Ramp up to 200 users
    { duration: '5m', target: 200 }, // Stay at 200 users
    { duration: '2m', target: 0 },   // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(99)<1500'], // 99% of requests must complete below 1.5s
    errors: ['rate<0.1'], // Error rate must be below 10%
  },
};

const BASE_URL = 'http://nginx';

export default function () {
  // Test health endpoint
  let healthResponse = http.get(`${BASE_URL}/health`);
  check(healthResponse, {
    'health check status is 200': (r) => r.status === 200,
    'health check response time < 500ms': (r) => r.timings.duration < 500,
  }) || errorRate.add(1);

  sleep(1);

  // Test webhook endpoint (simulate Telegram webhook)
  let webhookPayload = JSON.stringify({
    update_id: Math.floor(Math.random() * 1000000),
    message: {
      message_id: Math.floor(Math.random() * 1000),
      from: {
        id: Math.floor(Math.random() * 100000),
        first_name: 'TestUser',
        username: 'testuser' + Math.floor(Math.random() * 1000),
      },
      chat: {
        id: Math.floor(Math.random() * 100000),
        type: 'private',
      },
      date: Math.floor(Date.now() / 1000),
      text: '/start',
    },
  });

  let webhookResponse = http.post(`${BASE_URL}/webhook`, webhookPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  check(webhookResponse, {
    'webhook status is 200': (r) => r.status === 200,
    'webhook response time < 2000ms': (r) => r.timings.duration < 2000,
  }) || errorRate.add(1);

  sleep(2);
}