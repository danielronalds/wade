type ErrorResponse = {
  message: string;
};

const sessionPath = '/api/session';

const isErrorResponse = (value: unknown): value is ErrorResponse => {
  if (!value || typeof value !== 'object') {
    return false;
  }

  return typeof (value as Partial<ErrorResponse>).message === 'string';
};

const responseErrorMessage = async (response: Response, fallback: string) => {
  const text = await response.text();
  if (text.trim() === '') {
    return fallback;
  }

  try {
    const body: unknown = JSON.parse(text);
    if (isErrorResponse(body) && body.message.trim() !== '') {
      return body.message;
    }
  } catch {
    return text.trim();
  }

  return fallback;
};

export const closeProjectSession = async (project: string): Promise<void> => {
  const response = await fetch(`${sessionPath}/${encodeURIComponent(project)}`, { method: 'DELETE' });

  if (!response.ok) {
    throw new Error(await responseErrorMessage(response, `Session close failed with ${response.status}`));
  }
};
