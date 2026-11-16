import http from 'k6/http';
import { check, sleep } from 'k6';

// Step 2: Breaking Point / Stress Test
// Stepping load: 1,000 -> 1,500 -> 2,500 -> 3,500 RPS
// Used to discover the maximum throughput the Go server and Azure VM can handle before degradation.
export const options = {
  scenarios: {
    breaking_point_stress: {
      executor: 'ramping-arrival-rate',
      startRate: 500,
      timeUnit: '1s',
      preAllocatedVUs: 150,
      maxVUs: 1000,
      stages: [
        { duration: '30s', target: 1000 },  // Baseline: 1,000 RPS
        { duration: '45s', target: 1500 },  // Step up: 1,500 RPS
        { duration: '45s', target: 2500 },  // High stress: 2,500 RPS
        { duration: '45s', target: 3500 },  // Breaking search: 3,500 RPS
        { duration: '30s', target: 500 },   // Recovery: back to 500 RPS
      ],
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.10'],    // Track whether failure rate crosses 10%
    http_req_duration: ['p(95)<250'],  // Track latency degradation under heavy load
  },
};

const BASE_URL = __ENV.TARGET_URL || 'http://localhost:8080';

export function setup() {
  const codes = [];
  for (let i = 0; i < 5; i++) {
    const res = http.post(`${BASE_URL}/api/v1/urls`, JSON.stringify({
      original_url: `https://example.com/stress-test-${i}`,
      expires_in: '24h',
    }), {
      headers: { 'Content-Type': 'application/json' },
    });

    if (res.status === 201) {
      codes.push(JSON.parse(res.body).data.short_code);
    }
  }
  return { shortCodes: codes };
}

export default function (data) {
  // Select a random cached URL
  const code = data.shortCodes[Math.floor(Math.random() * data.shortCodes.length)];
  const res = http.get(`${BASE_URL}/${code}`, { redirects: 0 });

  check(res, {
    'status is 302': (r) => r.status === 302,
  });
}
