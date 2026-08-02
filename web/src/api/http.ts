type ProblemResponse = {
  detail: string;
};

const isProblemResponse = (value: unknown): value is ProblemResponse => {
  if (!value || typeof value !== 'object') {
    return false;
  }

  return typeof (value as Partial<ProblemResponse>).detail === 'string';
};

export const responseErrorMessage = async (response: Response, fallback: string) => {
  const text = await response.text();
  if (text.trim() === '') {
    return fallback;
  }

  try {
    const body: unknown = JSON.parse(text);
    if (isProblemResponse(body) && body.detail.trim() !== '') {
      return body.detail;
    }
  } catch {
    return text.trim();
  }

  return fallback;
};
