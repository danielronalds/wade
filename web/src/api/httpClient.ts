import { responseErrorMessage } from '@/api/http';

const emptyResponseStatuses = new Set([204, 205, 304]);

export const wadeFetch = async <T>(url: string, options?: RequestInit): Promise<T> => {
  const response = await fetch(url, options);
  if (!response.ok) {
    throw new Error(await responseErrorMessage(response, `Request failed with ${response.status}`));
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
