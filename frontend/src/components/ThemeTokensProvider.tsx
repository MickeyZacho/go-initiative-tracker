import React from "react";
import { ThemeProvider } from "@mui/material";
import { buildTheme, defaultTokens, type ThemeTokens } from "../theme";
import {
	PLAYGROUND_FONTS,
	sanitizeTokens,
	ThemeTokensContext,
	type ThemeSaveTarget,
} from "../lib/themeTokens";
import { apiGet, apiPost } from "../lib/http";

const STORAGE_KEY = "themePlayground.tokens";
// Already loaded by index.html.
const PRELOADED = new Set(["Atkinson Hyperlegible", "IBM Plex Sans"]);
// Wait for a pause in slider dragging / typing before saving to the server.
const SAVE_DELAY_MS = 800;

// The browser copy applies instantly on load (no flash of the default theme)
// and is the only copy for logged-out users.
const loadLocal = (): ThemeTokens => {
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		return raw ? sanitizeTokens(JSON.parse(raw)) : defaultTokens;
	} catch {
		return defaultTokens;
	}
};

const saveLocal = (tokens: ThemeTokens | null) => {
	try {
		if (tokens) localStorage.setItem(STORAGE_KEY, JSON.stringify(tokens));
		else localStorage.removeItem(STORAGE_KEY);
	} catch {
		// Not remembered in this browser; the account copy (if any) still is.
	}
};

const ensureFontLoaded = (family: string) => {
	if (PRELOADED.has(family)) return;
	const id = `theme-font-${family.replace(/\s+/g, "-")}`;
	if (document.getElementById(id)) return;
	const spec = PLAYGROUND_FONTS.find((f) => f.family === family)?.spec ?? "";
	const link = document.createElement("link");
	link.id = id;
	link.rel = "stylesheet";
	link.href = `https://fonts.googleapis.com/css2?family=${family.replace(/ /g, "+")}${spec ? `:${spec}` : ""}&display=swap`;
	document.head.appendChild(link);
};

// Builds the MUI theme from the user's tokens. Signed-in users' themes are
// saved to their account (GET/POST /user/theme) so they follow them across
// devices; everyone's is also kept in this browser.
export const ThemeTokensProvider: React.FC<{ children: React.ReactNode }> = ({
	children,
}) => {
	const [tokens, setTokensState] = React.useState(loadLocal);
	const [savedTo, setSavedTo] = React.useState<ThemeSaveTarget>("browser");
	const [saveError, setSaveError] = React.useState("");
	const saveTimer = React.useRef<number | undefined>(undefined);
	// The React Compiler memoizes this on tokens.
	const theme = buildTheme(tokens);

	// The account copy wins over the browser copy once it arrives.
	React.useEffect(() => {
		let mounted = true;
		apiGet<{ loggedIn?: boolean; theme?: unknown }>("/user/theme")
			.then((data) => {
				if (!mounted || !data.loggedIn) return;
				setSavedTo("account");
				if (data.theme) {
					const fromAccount = sanitizeTokens(data.theme);
					setTokensState(fromAccount);
					saveLocal(fromAccount);
				}
			})
			.catch(() => {
				// Offline or backend down: keep the browser copy.
			});
		return () => {
			mounted = false;
			window.clearTimeout(saveTimer.current);
		};
	}, []);

	React.useEffect(() => {
		ensureFontLoaded(tokens.headingFont);
		ensureFontLoaded(tokens.bodyFont);
	}, [tokens.headingFont, tokens.bodyFont]);

	// null = back to the app default.
	const persist = (next: ThemeTokens | null) => {
		saveLocal(next);
		if (savedTo !== "account") return;
		window.clearTimeout(saveTimer.current);
		saveTimer.current = window.setTimeout(() => {
			apiPost("/user/theme", { theme: next }, "Failed to save theme")
				.then(() => setSaveError(""))
				.catch((err) =>
					setSaveError(
						err instanceof Error ? err.message : "Failed to save theme",
					),
				);
		}, SAVE_DELAY_MS);
	};

	const setTokens = (update: (prev: ThemeTokens) => ThemeTokens) => {
		const next = update(tokens);
		setTokensState(next);
		persist(next);
	};
	const reset = () => {
		setTokensState(defaultTokens);
		persist(null);
	};

	return (
		<ThemeTokensContext
			value={{ tokens, setTokens, reset, savedTo, saveError }}
		>
			<ThemeProvider theme={theme} defaultMode="system">
				{tokens.customCss && <style>{tokens.customCss}</style>}
				{children}
			</ThemeProvider>
		</ThemeTokensContext>
	);
};
