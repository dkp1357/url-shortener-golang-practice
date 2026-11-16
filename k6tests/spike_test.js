import http from 'k6/http';
import { check } from 'k6';

// Step 2 (Alternative): Spike Test
// Simulates an abrupt flash mob: jumps from 100 RPS to 2,500 RPS in 5 seconds.
// Tests whether connection pooling, goroutine scheduling, and socket queues survive instant load spikes.
export const options = {
  scenarios: {
    instant_spike: {
      executor: 'ramping-arrival-rate',
      startRate: 100,
      timeUnit: '1s',
      preAllocatedVUs: 100,
      maxVUs: 800,
      stages: [
        { duration: '20s', target: 100 },   // Steady low baseline
        { duration: '5s',  target: 2500 },  // Sudden spike to 2,500 RPS
        { duration: '40s', target: 2500 },  // Sustain the surge
        { duration: '5s',  target: 100 },   // Immediate drop
        { duration: '20s', target: 100 },   // Verify system recovery
      ],
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.05'],    // Under 5% errors during the sudden spike
    http_req_duration: ['p(95)<300'],  // p95 latency under 300ms during spike
  },
};

const BASE_URL = __ENV.TARGET_URL || 'http://localhost:8080';

export function setup() {
  const res = http.post(`${BASE_URL}/api/v1/urls`, JSON.stringify({
    original_url: 'https://github.com',
    expires_in: '24h',
  }), {
    headers: { 'Content-Type': 'application/json' },
  });
  const body = JSON.parse(res.body);
  return { code: body.data.short_code };
}

export default function (data) {
  const res = http.get(`${BASE_URL}/${data.code}`, { redirects: 0 });
  check(res, {
    'status is 302': (r) => r.status === 302,
  });
}
