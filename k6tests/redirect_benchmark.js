import http from 'k6/http';
import { check } from 'k6';

// Step 1: Fix VU allocation with higher headroom (preAllocatedVUs: 100, maxVUs: 500)
// Prevents "Insufficient VUs" warnings during network/CPU jitter.
export const options = {
  scenarios: {
    constant_request_rate: {
      executor: 'constant-arrival-rate',
      rate: 1000,           // 1,000 requests per second
      timeUnit: '1s',
      duration: '1m',
      preAllocatedVUs: 100, // increased from 50 to allocate enough workers upfront
      maxVUs: 500,          // increased from 200 to absorb network/latency spikes without dropped iterations
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],    // Error rate must remain under 1%
    http_req_duration: ['p(95)<60'],   // 95% of requests should complete under 60ms
  },
};

const BASE_URL = __ENV.TARGET_URL || 'http://localhost:8080';

export function setup() {
  const payload = JSON.stringify({
    original_url: 'https://news.ycombinator.com',
  });
  const res = http.post(`${BASE_URL}/api/v1/urls`, payload, {
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
