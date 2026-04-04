import { useState, useCallback } from "react";
import { apiUrl } from "../lib/api";
import { parseJsonResponse } from "../lib/http";
import type { LedgerEntry } from "../components/CharacterList";

export function useCombatLog() {
	const [ledgerEntries, setLedgerEntries] = useState<LedgerEntry[]>([]);
	const [error, setError] = useState<string>("");

	const fetchLedger = useCallback(async (encId: number) => {
		if (!encId) {
			setLedgerEntries([]);
			return;
		}
		try {
			const response = await fetch(
				apiUrl(`/api/encounters/ledger?encounter_id=${encId}`),
				{ credentials: "include" },
			);
			const payload = await parseJsonResponse<{
				status?: string;
				entries?: LedgerEntry[];
				message?: string;
			}>(response);
			if (!response.ok || payload.status !== "success") {
				throw new Error(payload.message || "Failed to load combat log");
			}
			setLedgerEntries(
				Array.isArray(payload.entries) ? payload.entries : [],
			);
		} catch {
			setLedgerEntries([]);
		}
	}, []);

	return {
		ledgerEntries,
		setLedgerEntries,
		fetchLedger,
		error,
	};
}
