let authToken: string | null = null;

export function setAuthToken(token: string | null) {
  authToken = token;
}

export async function apiFetch(url: string, options?: RequestInit) {
  const headers: Record<string, string> = { // Explicitly type headers
    ...options?.headers as Record<string, string>, // Cast options.headers
  };

  if (authToken) {
    headers['Authorization'] = `Bearer ${authToken}`;
  }

  const response = await fetch(url, {
    ...options,
    headers,
  });
  
  if (!response.ok) {
    let errorMessage = `HTTP error! Status: ${response.status}`;
    try {
      const errorBody = await response.json();
      if (errorBody.message) {
        errorMessage = errorBody.message;
      } else if (errorBody.error) {
        errorMessage = errorBody.error;
      }
    } catch (e) {
      // If response is not JSON, use the status text or a generic message
      errorMessage = response.statusText || 'An unexpected error occurred';
    }
    console.error(`API Error: ${url} - ${errorMessage}`);
    throw new Error(errorMessage);
  }

  const contentType = response.headers.get('content-type');
  // If there's no content or content type, return null
  if (response.status === 204 || !contentType) {
    return null;
  }

  if (contentType.includes('application/json')) {
    // Check if the response body is empty before parsing as JSON
    const text = await response.text();
    return text ? JSON.parse(text) : null;
  } else if (contentType.includes('text/')) {
    // If it's text, return the text directly
    return response.text();
  } else {
    // For other content types, return null or handle as needed
    return null;
  }
}

export async function apiFetchFile(url: string, options?: RequestInit): Promise<Blob> {
  const headers: Record<string, string> = {
    ...options?.headers as Record<string, string>,
  };

  if (authToken) {
    headers['Authorization'] = `Bearer ${authToken}`;
  }

  const response = await fetch(url, {
    ...options,
    headers,
  });

  if (!response.ok) {
    let errorMessage = `HTTP error! Status: ${response.status}`;
    try {
      const errorBody = await response.json();
      if (errorBody.message) {
        errorMessage = errorBody.message;
      } else if (errorBody.error) {
        errorMessage = errorBody.error;
      }
    } catch (e) {
      errorMessage = response.statusText || 'An unexpected error occurred';
    }
    console.error(`API Error: ${url} - ${errorMessage}`);
    throw new Error(errorMessage);
  }

  return response.blob();
}

// Add this function
export function handleApiError(error: unknown): string {
  if (error instanceof Error) {
    console.error('API Error:', error.message);
    return error.message;
  }
  console.error('Unknown API error:', error);
  return 'An unknown error occurred';
}