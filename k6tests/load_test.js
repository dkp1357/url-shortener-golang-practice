import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const redirectFailures = new Rate('redirect_failures');
const shortenFailures = new Rate('shorten_failures');
const redirectLatency = new Trend('redirect_duration');

export const options = {
    scenarios: {
        // Scenario 1: Mixed traffic ramping up to 200 concurrent users
        url_service_stress: {
            executor: 'ramping-vus',
            startVUs: 0,
            stages: [
                { duration: '30s', target: 50 },  // Warm up
                { duration: '1m', target: 150 }, // Ramp to high load
                { duration: '1m', target: 150 }, // Sustain peak load
                { duration: '30s', target: 0 },   // Cool down
            ],
            gracefulRampDown: '10s',
        },
    },
    thresholds: {
        http_req_failed: ['rate<0.05'],            // Overall errors < 5%
        'http_req_duration{type:redirect}': ['p(95)<100'], // 95% of cached redirects under 100ms
        'http_req_duration{type:shorten}': ['p(95)<300'],  // 95% of shorten creation under 300ms
    },
};

const BASE_URL = __ENV.TARGET_URL || 'http://localhost:8080';

// Global shared short codes created during setup
export function setup() {
    console.log(`Setting up test against: ${BASE_URL}`);

    // Pre-seed 10 short URLs to populate Redis cache for read benchmarks
    const preSeededCodes = [];
    for (let i = 0; i < 10; i++) {
        const payload = JSON.stringify({
            original_url: `https://example.com/seeded-page-${i}`,
            expires_in: '24h',
        });

        const res = http.post(`${BASE_URL}/api/v1/urls`, payload, {
            headers: { 'Content-Type': 'application/json' },
        });

        if (res.status === 201) {
            const body = JSON.parse(res.body);
            preSeededCodes.push(body.data.short_code);
        }
    }

    // Ensure Redis has warmed up for the first key
    if (preSeededCodes.length > 0) {
        http.get(`${BASE_URL}/${preSeededCodes[0]}`, { redirects: 0 });
    }

    return { shortCodes: preSeededCodes };
}

export default function (data) {
    const rand = Math.random();

    // 80% of traffic: Read / Redirects (GET /{code}) -> Hits Redis / Cache-Aside
    if (rand < 0.8 && data.shortCodes && data.shortCodes.length > 0) {
        const code = data.shortCodes[Math.floor(Math.random() * data.shortCodes.length)];

        // Disable auto-following redirects so we inspect the 302 Found response directly
        const res = http.get(`${BASE_URL}/${code}`, {
            redirects: 0,
            tags: { type: 'redirect' },
        });

        redirectLatency.add(res.timings.duration);
        const success = check(res, {
            'redirect is 302': (r) => r.status === 302,
            'has Location header': (r) => r.headers['Location'] !== undefined,
        });
        redirectFailures.add(!success);
    }
    // 20% of traffic: Write / Shorten URL (POST /api/v1/urls) -> Hits PostgreSQL + Redis Invalidation
    else {
        const payload = JSON.stringify({
            original_url: `https://example.com/test-${__VU}-${__ITER}-${Date.now()}`,
            expires_in: '12h',
        });

        const res = http.post(`${BASE_URL}/api/v1/urls`, payload, {
            headers: { 'Content-Type': 'application/json' },
            tags: { type: 'shorten' },
        });

        const success = check(res, {
            'shorten is 201': (r) => r.status === 201,
            'has short_code in response': (r) => {
                try {
                    const body = JSON.parse(r.body);
                    return body.data && body.data.short_code !== undefined;
                } catch (_) {
                    return false;
                }
            },
        });
        shortenFailures.add(!success);
    }

    sleep(0.1); // Small think time between requests
}

export function teardown(data) {
    console.log(`Test completed. Pre-seeded codes used: ${data.shortCodes.length}`);
}