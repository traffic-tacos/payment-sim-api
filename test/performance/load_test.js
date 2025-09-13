import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const paymentIntentCreationTime = new Trend('payment_intent_creation_time');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 10 },   // Warm up
    { duration: '1m', target: 50 },    // Load testing
    { duration: '2m', target: 100 },   // Peak load
    { duration: '1m', target: 200 },   // Stress testing
    { duration: '30s', target: 0 },    // Cool down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests should be below 500ms
    http_req_failed: ['rate<0.1'],    // Error rate should be below 10%
    errors: ['rate<0.1'],             // Custom error rate
    payment_intent_creation_time: ['p(95)<200'], // Payment intent creation should be fast
  },
  tags: {
    test_type: 'load_test',
  },
};

// Base URL
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Test data
const scenarios = ['approve', 'fail', 'delay', 'random'];
const webhookUrls = [
  'http://httpbin.org/post',
  'https://webhook.site/test',
  'http://localhost:8081/webhook', // Mock webhook endpoint
];

function generateIdempotencyKey() {
  return `test-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

function getRandomScenario() {
  return scenarios[Math.floor(Math.random() * scenarios.length)];
}

function getRandomWebhookUrl() {
  return webhookUrls[Math.floor(Math.random() * webhookUrls.length)];
}

export default function () {
  // Test payment intent creation
  const idempotencyKey = generateIdempotencyKey();

  const payload = {
    reservation_id: `rsv_${Date.now()}_${__VU}_${__ITER}`,
    amount: Math.floor(Math.random() * 100000) + 10000, // 10k ~ 110k
    scenario: getRandomScenario(),
    webhook_url: getRandomWebhookUrl(),
    metadata: {
      test_run: true,
      vu: __VU,
      iteration: __ITER,
      timestamp: new Date().toISOString(),
    },
  };

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Idempotency-Key': idempotencyKey,
    },
    timeout: '10s',
  };

  const startTime = new Date().getTime();
  const response = http.post(`${BASE_URL}/v1/sim/intent`, JSON.stringify(payload), params);
  const endTime = new Date().getTime();

  // Record custom metric
  paymentIntentCreationTime.add(endTime - startTime);

  // Check response
  const checkResult = check(response, {
    'status is 200': (r) => r.status === 200,
    'response time < 500ms': (r) => r.timings.duration < 500,
    'has payment_intent_id': (r) => r.json().hasOwnProperty('payment_intent_id'),
    'has status': (r) => r.json().hasOwnProperty('status'),
    'has next': (r) => r.json().hasOwnProperty('next'),
    'next is webhook': (r) => r.json().next === 'webhook',
  });

  // Record error rate
  errorRate.add(!checkResult);

  // Log failures
  if (!checkResult) {
    console.log(`Request failed: ${response.status} - ${response.body}`);
  }

  // Small delay between requests
  sleep(0.1);
}

// Setup function - runs before the test starts
export function setup() {
  console.log('🚀 Starting Payment Simulator API Load Test');
  console.log(`📍 Target URL: ${BASE_URL}`);

  // Health check
  const healthResponse = http.get(`${BASE_URL}/healthz`);
  if (healthResponse.status !== 200) {
    console.error(`❌ Health check failed: ${healthResponse.status}`);
    return;
  }

  console.log('✅ Health check passed');
  return { timestamp: new Date().toISOString() };
}

// Teardown function - runs after the test completes
export function teardown(data) {
  console.log('🏁 Load test completed');
  console.log(`⏰ Test started at: ${data.timestamp}`);
  console.log(`⏰ Test completed at: ${new Date().toISOString()}`);
}

// Handle summary - custom summary output
export function handleSummary(data) {
  const summary = {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'test/performance/results.json': JSON.stringify(data, null, 2),
    'test/performance/summary.html': htmlReport(data),
  };

  return summary;
}

function textSummary(data, options) {
  return `
📊 Payment Simulator API Load Test Summary
===============================================

Test Duration: ${data.metrics.iteration_duration.values.avg}ms avg iteration
Total Requests: ${data.metrics.http_reqs.values.count}
Failed Requests: ${data.metrics.http_req_failed.values.rate * 100}%

🚀 HTTP Request Duration:
  - Average: ${Math.round(data.metrics.http_req_duration.values.avg)}ms
  - 95th percentile: ${Math.round(data.metrics.http_req_duration.values['p(95)']}ms
  - 99th percentile: ${Math.round(data.metrics.http_req_duration.values['p(99)']}ms

💳 Payment Intent Creation Time:
  - Average: ${Math.round(data.metrics.payment_intent_creation_time.values.avg)}ms
  - 95th percentile: ${Math.round(data.metrics.payment_intent_creation_time.values['p(95)']}ms

❌ Error Rates:
  - HTTP errors: ${data.metrics.http_req_failed.values.rate * 100}%
  - Custom errors: ${data.metrics.errors.values.rate * 100}%

📈 Throughput: ${Math.round(data.metrics.http_reqs.values.rate)} requests/second

✅ Checks:
  - Status is 200: ${data.metrics.checks.values.rate * 100}%
  `;
}

function htmlReport(data) {
  return `
<!DOCTYPE html>
<html>
<head>
    <title>Payment Simulator API Load Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .metric { background: #f5f5f5; padding: 10px; margin: 10px 0; border-radius: 5px; }
        .success { color: green; }
        .warning { color: orange; }
        .error { color: red; }
        h1, h2 { color: #333; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <h1>Payment Simulator API Load Test Report</h1>

    <div class="metric">
        <h2>📊 Summary</h2>
        <p><strong>Total Requests:</strong> ${data.metrics.http_reqs.values.count}</p>
        <p><strong>Duration:</strong> ${Math.round(data.metrics.iteration_duration.values.avg)}ms avg</p>
        <p><strong>Throughput:</strong> ${Math.round(data.metrics.http_reqs.values.rate)} req/sec</p>
    </div>

    <div class="metric">
        <h2>🚀 Performance</h2>
        <table>
            <tr><th>Metric</th><th>Value</th></tr>
            <tr><td>Average Response Time</td><td>${Math.round(data.metrics.http_req_duration.values.avg)}ms</td></tr>
            <tr><td>95th Percentile</td><td>${Math.round(data.metrics.http_req_duration.values['p(95)']}ms</td></tr>
            <tr><td>99th Percentile</td><td>${Math.round(data.metrics.http_req_duration.values['p(99)']}ms</td></tr>
        </table>
    </div>

    <div class="metric">
        <h2>❌ Error Rates</h2>
        <p class="${data.metrics.http_req_failed.values.rate > 0.1 ? 'error' : 'success'}">
            HTTP Errors: ${(data.metrics.http_req_failed.values.rate * 100).toFixed(2)}%
        </p>
        <p class="${data.metrics.errors.values.rate > 0.1 ? 'error' : 'success'}">
            Custom Errors: ${(data.metrics.errors.values.rate * 100).toFixed(2)}%
        </p>
    </div>

    <div class="metric">
        <h2>💳 Payment Intent Creation</h2>
        <p><strong>Average Time:</strong> ${Math.round(data.metrics.payment_intent_creation_time.values.avg)}ms</p>
        <p><strong>95th Percentile:</strong> ${Math.round(data.metrics.payment_intent_creation_time.values['p(95)']}ms</p>
    </div>
</body>
</html>
  `;
}
