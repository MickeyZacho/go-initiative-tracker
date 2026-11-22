import { useEffect, useState } from "react";
import { apiGet, apiGetArray } from "../lib/http";

export interface StatBlock {
	Strength: number;
	Dexterity: number;
	Constitution: number;
	Intelligence: number;
	Wisdom: number;
	Charisma: number;
}

export interface NpcTemplate {
	ID: number;
	Name: string;
	Description: string;
	BaseStats: StatBlock;
	ArmorClass: number;
	MaxHP: number;
	OwnerID?: string;
	/** Where an imported template came from ("srd-5.2"); empty for app-made ones. */
	Source?: string;
	/** From an imported template's details; empty/null for app-made ones. */
	Type?: string;
	Size?: string;
	CR?: number | null;
}

/** Optional filters. Type, size and CR only match imported (SRD) monsters. */
export interface NpcTemplateFilters {
	type?: string;
	size?: string;
	minCR?: number;
	maxCR?: number;
	source?: "" | "custom" | "srd";
	sort?: "" | "name" | "cr";
}

/** Page size; the backend caps it too. */
export const NPC_SEARCH_LIMIT = 24;

/** Standard 5e challenge ratings, for CR pickers. */
export const CR_VALUES = [0, 0.125, 0.25, 0.5, ...Array.from({ length: 30 }, (_, i) => i + 1)];

export function formatCR(cr: number): string {
	if (cr === 0.125) return "1/8";
	if (cr === 0.25) return "1/4";
	if (cr === 0.5) return "1/2";
	return String(cr);
}

function buildParams(q: string, f: NpcTemplateFilters): URLSearchParams {
	const p = new URLSearchParams();
	if (q) p.set("q", q);
	if (f.type) p.set("type", f.type);
	if (f.size) p.set("size", f.size);
	if (f.minCR !== undefined) p.set("min_cr", String(f.minCR));
	if (f.maxCR !== undefined) p.set("max_cr", String(f.maxCR));
	if (f.source) p.set("source", f.source);
	if (f.sort) p.set("sort", f.sort);
	p.set("limit", String(NPC_SEARCH_LIMIT));
	return p;
}

/**
 * Server-side, paged template search. With no query or filters it returns
 * everything (app-made templates first), one page at a time; loadMore appends
 * the next page. Typing is debounced so each keystroke doesn't hit the API.
 */
export function useNpcTemplateSearch(query: string, filters: NpcTemplateFilters = {}) {
	const [results, setResults] = useState<NpcTemplate[]>([]);
	const [loading, setLoading] = useState(false);
	const [hasMore, setHasMore] = useState(false);
	const [error, setError] = useState("");
	const [reloadKey, setReloadKey] = useState(0);
	const q = query.trim();
	// A string key so a fresh filters object each render doesn't refetch.
	const params = buildParams(q, filters).toString();

	const fetchPage = async (offset: number) => {
		const data = await apiGetArray<NpcTemplate>(
			`/npcs/templates?${params}&offset=${offset}`,
		);
		setHasMore(data.length >= NPC_SEARCH_LIMIT);
		return data;
	};

	useEffect(() => {
		let active = true;
		const timer = setTimeout(
			async () => {
				setLoading(true);
				try {
					const data = await apiGetArray<NpcTemplate>(
						`/npcs/templates?${params}&offset=0`,
					);
					if (active) {
						setResults(data);
						setHasMore(data.length >= NPC_SEARCH_LIMIT);
						setError("");
					}
				} catch (err) {
					if (active) {
						setResults([]);
						setHasMore(false);
						setError(err instanceof Error ? err.message : "Failed to load npcs");
					}
				} finally {
					if (active) setLoading(false);
				}
			},
			q ? 250 : 0,
		);
		return () => {
			active = false;
			clearTimeout(timer);
		};
	}, [params, q, reloadKey]);

	const loadMore = async () => {
		if (loading || !hasMore) return;
		setLoading(true);
		try {
			const data = await fetchPage(results.length);
			setResults((prev) => {
				const seen = new Set(prev.map((n) => n.ID));
				return [...prev, ...data.filter((n) => !seen.has(n.ID))];
			});
		} catch (err) {
			setError(err instanceof Error ? err.message : "Failed to load npcs");
		} finally {
			setLoading(false);
		}
	};

	return {
		results,
		/** Local edits (inline editing on the NPC page) before they're saved. */
		setResults,
		loading,
		error,
		/** True when the last page was full, so more may exist. */
		hasMore,
		loadMore,
		reload: () => setReloadKey((k) => k + 1),
	};
}

export interface NpcTemplateFacets {
	Types: string[];
	Sizes: string[];
}

/** Distinct creature types and sizes in the data, for filter dropdowns. */
export function useNpcTemplateFacets() {
	const [facets, setFacets] = useState<NpcTemplateFacets>({ Types: [], Sizes: [] });
	useEffect(() => {
		let active = true;
		apiGet<NpcTemplateFacets>("/npcs/templates/facets")
			.then((f) => {
				if (active) setFacets({ Types: f.Types ?? [], Sizes: f.Sizes ?? [] });
			})
			.catch(() => {});
		return () => {
			active = false;
		};
	}, []);
	return facets;
}
