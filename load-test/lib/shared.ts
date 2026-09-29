import http from 'k6/http';
import type { Options } from 'k6/options';
import type { RefinedResponse, ResponseType } from 'k6/http';

// same load
export const loadProfile: Options = {
  stages: [
    { duration: '10s', target: 5 },
    { duration: '20s', target: 5 },
    { duration: '10s', target: 15 },
    { duration: '20s', target: 15 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.01'],
  },
};

// The one call both versions genuinely share — same endpoint, same method.
export function submitGeneratePdf(baseUrl: string): RefinedResponse<ResponseType> {
  return http.post(`${baseUrl}/generate-pdf`, null, { timeout: '35s' });
}


