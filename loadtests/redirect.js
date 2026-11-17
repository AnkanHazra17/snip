import http from 'k6/http';
import { check } from 'k6';

export const options = {
  vus: 50,              // 50 virtual users at the same time
  duration: '30s',
};

const CODE = __ENV.CODE;  // passed in from the command line

export default function () {
  const res = http.get(`http://localhost:8080/${CODE}`, { redirects: 0 });
  check(res, { 'is 302': (r) => r.status === 302 });
}