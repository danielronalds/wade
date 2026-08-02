import { responseErrorMessage } from '@/api/http';

const emptyResponseStatuses = new Set([204, 205, 304]);

export class WadeHTTPError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = 'WadeHTTPError';
    this.status = status;
  }
}

export const wadeFetch = async <T>(url: string, options?: RequestInit): Promise<T> => {
  const response = await fetch(url, options);
  if (!response.ok) {
    const message = await responseErrorMessage(response, `Request failed with ${response.status}`);
    throw new WadeHTTPError(response.status, message);
  }

  if (emptyResponseStatuses.has(response.status)) {
    return undefined as T;
  }

  const text = await response.text();
  if (text.trim() === '') {
    return undefined as T;
  }

  return JSON.parse(text) as T;
};
