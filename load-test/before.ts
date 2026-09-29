import { check } from 'k6';
import { loadProfile, submitGeneratePdf } from './lib/shared.ts';

const BASE_URL: string = __ENV.BASE_URL || 'http://localhost:30080';

export const options = loadProfile;

export default function (): void {
  const res = submitGeneratePdf(BASE_URL);

  check(res, {
    'status is 200': (r) => r.status === 200,
    'got a pdf': (r) => r.headers['Content-Type'] === 'application/pdf',
  });
}