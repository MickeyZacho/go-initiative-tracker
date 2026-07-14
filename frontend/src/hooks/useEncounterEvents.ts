import { useEffect, useRef } from "react";

// useEncounterEvents subscribes to the encounter's Server-Sent Events stream and
// invokes onChange whenever the backend broadcasts a change (HP, turn, roster,
// ledger). The stream carries a nudge, not state, so onChange should re-fetch.
//
// The EventSource is same-origin (via the /api Vite/Caddy proxy), so the Discord
// session cookie is sent automatically and it reconnects on its own if dropped.
// onChange is debounced so a burst of events (e.g. a quick action emits a
// character + a ledger event) collapses into a single refresh.
export function useEncounterEvents(
	encounterId: number,
	onChange: () => void,
) {
	// Keep the latest onChange in a ref so re-renders don't churn the EventSource.
	const onChangeRef = useRef(onChange);
	onChangeRef.current = onChange;

	useEffect(() => {
		if (!encounterId) {
			return;
		}
		const source = new EventSource(
			`/api/encounters/events?encounter_id=${encounterId}`,
		);

		let debounce: ReturnType<typeof setTimeout> | undefined;
		source.onmessage = () => {
			if (debounce) clearTimeout(debounce);
			debounce = setTimeout(() => onChangeRef.current(), 150);
		};

		return () => {
			if (debounce) clearTimeout(debounce);
			source.close();
		};
	}, [encounterId]);
}
