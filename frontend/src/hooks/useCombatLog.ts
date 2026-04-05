import { useState, useCallback } from "react";
import { parseJsonResponse } from "../lib/http";

// Locally define LedgerEntry interface for combat log entries
export interface LedgerEntry {
	id: number;
	encounter_id: number;
	actor_id: number;
	actor_name: string; // Added for easier display in the log
	target_id: number | null;
	target_name: string | null; // Added for easier display in the log
	action_type: string;
	hp_change: number;
	description: string;
	created_at: string; // ISO timestamp
}

export function useCombatLog() {
	const [ledgerEntries, setLedgerEntries] = useState<LedgerEntry[]>([]);

	const fetchLedger = useCallback(async (encId: number) => {
		if (!encId) {
			setLedgerEntries([]);
			return;
		}
		try {
			const response = await fetch(
				`/api/encounters/ledger?encounter_id=${encId}`,
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
	};
}
