// Centralized helpers for talking to the backend API.
//
// Every request goes through the Vite dev proxy at `/api` (see vite.config.ts),
// includes credentials so the Discord session cookie is sent, and parses the
// response defensively so an HTML error page never blows up as a JSON parse.

const API_BASE = "/api";

export async function parseJsonResponse<T = unknown>(
	response: Response,
): Promise<T> {
	const text = await response.text();
	if (!text) {
		return null as T;
	}
	try {
		return JSON.parse(text) as T;
	} catch {
		const preview = text.slice(0, 120).replace(/\s+/g, " ");
		throw new Error(`Expected JSON but received: ${preview}`);
	}
}

/** GET a JSON array, returning `[]` when the payload is missing or malformed. */
export async function apiGetArray<T>(path: string): Promise<T[]> {
	const response = await fetch(`${API_BASE}${path}`, {
		credentials: "include",
	});
	if (!response.ok) {
		throw new Error(`Request to ${path} failed (${response.status})`);
	}
	const payload = await parseJsonResponse<unknown>(response);
	return Array.isArray(payload) ? (payload as T[]) : [];
}

/** GET a JSON object. Throws on a non-2xx response. */
export async function apiGet<T>(path: string): Promise<T> {
	const response = await fetch(`${API_BASE}${path}`, {
		credentials: "include",
	});
	if (!response.ok) {
		throw new Error(`Request to ${path} failed (${response.status})`);
	}
	return parseJsonResponse<T>(response);
}

interface ApiEnvelope {
	status?: string;
	message?: string;
}

/**
 * POST a JSON body and enforce the `{ status: "success" }` envelope the API
 * returns for mutations. Throws with the server-provided message (or
 * `fallbackError`) when the request fails or the envelope is not successful.
 */
export async function apiPost<T extends ApiEnvelope = ApiEnvelope>(
	path: string,
	body?: unknown,
	fallbackError = "Request failed",
): Promise<T> {
	const response = await fetch(`${API_BASE}${path}`, {
		method: "POST",
		credentials: "include",
		headers: { "Content-Type": "application/json" },
		body: body === undefined ? undefined : JSON.stringify(body),
	});
	const payload = await parseJsonResponse<T>(response);
	if (!response.ok || payload?.status !== "success") {
		throw new Error(payload?.message || fallbackError);
	}
	return payload;
}
