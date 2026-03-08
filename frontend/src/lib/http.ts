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
