import { afterEach, describe, expect, it, vi } from "vitest";
import { apiGet, apiGetArray, apiPost, parseJsonResponse } from "./http";

function jsonResponse(body: unknown, init?: ResponseInit): Response {
	return new Response(JSON.stringify(body), {
		status: 200,
		headers: { "Content-Type": "application/json" },
		...init,
	});
}

afterEach(() => {
	vi.restoreAllMocks();
});

describe("parseJsonResponse", () => {
	it("returns null for an empty body", async () => {
		expect(await parseJsonResponse(new Response(""))).toBeNull();
	});

	it("parses a JSON body", async () => {
		const parsed = await parseJsonResponse<{ a: number }>(
			jsonResponse({ a: 1 }),
		);
		expect(parsed).toEqual({ a: 1 });
	});

	it("throws a readable error on non-JSON", async () => {
		await expect(
			parseJsonResponse(new Response("<html>Boom</html>")),
		).rejects.toThrow(/Expected JSON but received/);
	});
});

describe("apiGetArray", () => {
	it("returns the array payload", async () => {
		vi.stubGlobal(
			"fetch",
			vi.fn().mockResolvedValue(jsonResponse([1, 2, 3])),
		);
		expect(await apiGetArray<number>("/things")).toEqual([1, 2, 3]);
	});

	it("coerces a non-array payload to []", async () => {
		vi.stubGlobal(
			"fetch",
			vi.fn().mockResolvedValue(jsonResponse({ not: "an array" })),
		);
		expect(await apiGetArray("/things")).toEqual([]);
	});

	it("throws with the status code on a non-2xx response", async () => {
		vi.stubGlobal(
			"fetch",
			vi.fn().mockResolvedValue(jsonResponse([], { status: 500 })),
		);
		await expect(apiGetArray("/things")).rejects.toThrow(/500/);
	});

	it("sends credentials so the session cookie rides along", async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse([]));
		vi.stubGlobal("fetch", fetchMock);
		await apiGetArray("/things");
		expect(fetchMock).toHaveBeenCalledWith(
			"/api/things",
			expect.objectContaining({ credentials: "include" }),
		);
	});
});

describe("apiGet", () => {
	it("returns the object payload", async () => {
		vi.stubGlobal(
			"fetch",
			vi.fn().mockResolvedValue(jsonResponse({ loggedIn: true })),
		);
		expect(await apiGet<{ loggedIn: boolean }>("/me")).toEqual({
			loggedIn: true,
		});
	});
});

describe("apiPost", () => {
	it("returns the payload on a success envelope", async () => {
		vi.stubGlobal(
			"fetch",
			vi.fn().mockResolvedValue(jsonResponse({ status: "success" })),
		);
		expect(await apiPost("/save", { a: 1 })).toEqual({ status: "success" });
	});

	it("throws the server message when the envelope is not success", async () => {
		vi.stubGlobal(
			"fetch",
			vi.fn().mockResolvedValue(
				jsonResponse({ status: "error", message: "Nope" }),
			),
		);
		await expect(apiPost("/save")).rejects.toThrow("Nope");
	});

	it("falls back to the provided error when there is no message", async () => {
		vi.stubGlobal(
			"fetch",
			vi.fn().mockResolvedValue(
				jsonResponse({ status: "error" }, { status: 500 }),
			),
		);
		await expect(apiPost("/save", undefined, "Save failed")).rejects.toThrow(
			"Save failed",
		);
	});
});
