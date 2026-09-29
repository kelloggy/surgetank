import http from 'k6/http';
import { check, sleep } from 'k6';
import { loadProfile, submitGeneratePdf } from './lib/shared.ts';

const BASE_URL: string = __ENV.BASE_URL || 'http://localhost:30081';

export const options = loadProfile;

export default function (): void {
  const submitRes = submitGeneratePdf(BASE_URL);
  const submitOk = check(submitRes, {
    'submit accepted': (r) => r.status === 200,
  });
  if (!submitOk) return;

  const jobId: string = submitRes.json('job_id') as string;

  let finalStatus = '';
  for (let i = 0; i < 20; i++) {
    sleep(1);
    const statusRes = http.get(`${BASE_URL}/status/${jobId}`);
    const status = statusRes.json('status') as string;
    if (status === 'done' || status === 'failed') {
      finalStatus = status;
      break;
    }
  }

  check(null, {
    'job completed (not stuck pending)': () => finalStatus === 'done',
  });

  if (finalStatus !== 'done') return;

  const resultRes = http.get(`${BASE_URL}/result/${jobId}`);
  check(resultRes, {
    'result is a pdf': (r) => r.headers['Content-Type'] === 'application/pdf',
  });
}