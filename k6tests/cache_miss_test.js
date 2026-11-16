import http from 'k6/http';
import { check } from 'k6';

// Step 3: Cache-Miss & Database Stress Test
// Generates random non-existent short codes (404s).
// Bypasses Redis hits and forces the Go backend to query PostgreSQL every single request.
// Tests pgx connection pool saturation, PostgreSQL query execution, and database disk/CPU impact.
export const options = {
  scenarios: {
    db_cache_miss: {
      executor: 'ramping-arrival-rate',
      startRate: 100,
      timeUnit: '1s',
      preAllocatedVUs: 100,
      maxVUs: 500,
      stages: [
        { duration: '20s', target: 300 },   // Warm up with 300 DB misses/sec
        { duration: '40s', target: 800 },   // Push DB to 800 queries/sec
        { duration: '30s', target: 1200 },  // High pressure: 1,200 DB queries/sec
        { duration: '20s', target: 100 },   // Ramp down
      ],
    },
  },
  thresholds: {
    // 404 is the expected business response for non-existent links;
    // this threshold catches actual 500 Internal Server Errors / DB connection drops.
    http_req_failed: ['rate<0.02'],
    http_req_duration: ['p(95)<150'], // Evaluates DB index search latency under load
  },
};

const BASE_URL = __ENV.TARGET_URL || 'http://localhost:8080';

// Helper to generate dynamic random 6-character short codes that don't exist in Redis
function randomString(length = 6) {
  const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
  let result = '';
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return result;
}

export default function () {
  // Query a brand new random code every iteration -> guarantees Redis cache miss -> hits Postgres
  const nonExistentCode = `miss_${randomString(8)}`;

  const res = http.get(`${BASE_URL}/${nonExistentCode}`, {
    redirects: 0,
    tags: { type: 'cache_miss' },
  });

  check(res, {
    'returns 404 Not Found': (r) => r.status === 404,
  });
}
