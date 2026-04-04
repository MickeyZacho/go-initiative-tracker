const API_BASE_URL =
	// (import.meta.env.VITE_API_BASE_URL as string | undefined) ??
	"http://localhost:8080";

export function apiUrl(path: string): string {
	if (!path.startsWith("/")) {
		return `${API_BASE_URL}/${path}`;
	}
	return `${API_BASE_URL}${path}`;
}
