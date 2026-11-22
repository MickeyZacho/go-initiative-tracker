import { createContext, useContext } from "react";
import { defaultTokens, type SchemeTokens, type ThemeTokens } from "../theme";

// Fonts the theme panel offers, with the Google Fonts axis spec to request
// ("" = the single regular weight, for display faces that only have one).
export const PLAYGROUND_FONTS: { family: string; spec: string }[] = [
	{ family: "Cinzel", spec: "wght@400..900" },
	{ family: "Cinzel Decorative", spec: "wght@400;700;900" },
	{ family: "Uncial Antiqua", spec: "" },
	{ family: "MedievalSharp", spec: "" },
	{ family: "Pirata One", spec: "" },
	{ family: "IM Fell English", spec: "" },
	{ family: "Playfair Display", spec: "wght@400..900" },
	{ family: "EB Garamond", spec: "wght@400..800" },
	{ family: "Lora", spec: "wght@400..700" },
	{ family: "Inter", spec: "wght@400..700" },
	{ family: "Roboto", spec: "wght@400..700" },
	{ family: "Open Sans", spec: "wght@400..700" },
	{ family: "Source Sans 3", spec: "wght@400..700" },
	{ family: "Nunito", spec: "wght@400..700" },
	{ family: "Lato", spec: "wght@400;700" },
	{ family: "IBM Plex Sans", spec: "wght@400;500;600;700" },
	{ family: "Atkinson Hyperlegible", spec: "wght@400;700" },
];

const HEX = /^#[0-9a-f]{6}$/i;
const FONT_NAMES = new Set(PLAYGROUND_FONTS.map((f) => f.family));
export const MAX_CUSTOM_CSS = 20_000;

const clamp = (v: unknown, min: number, max: number, fallback: number) =>
	typeof v === "number" && Number.isFinite(v)
		? Math.min(max, Math.max(min, Math.round(v)))
		: fallback;

const asRecord = (v: unknown) =>
	(v && typeof v === "object" ? v : {}) as Record<string, unknown>;

const scheme = (v: unknown, fallback: SchemeTokens): SchemeTokens => {
	const src = asRecord(v);
	const out = { ...fallback };
	for (const key of Object.keys(fallback) as (keyof SchemeTokens)[]) {
		const c = src[key];
		if (typeof c === "string" && HEX.test(c)) out[key] = c;
	}
	return out;
};

// sanitizeTokens turns anything (localStorage, the server, an older shape)
// into valid tokens, field by field, falling back to the defaults. MUI throws
// on a malformed colour, so nothing reaches buildTheme unchecked.
export const sanitizeTokens = (v: unknown): ThemeTokens => {
	const src = asRecord(v);
	const font = (f: unknown, fallback: string) =>
		typeof f === "string" && FONT_NAMES.has(f) ? f : fallback;
	return {
		light: scheme(src.light, defaultTokens.light),
		dark: scheme(src.dark, defaultTokens.dark),
		borderRadius: clamp(src.borderRadius, 0, 24, defaultTokens.borderRadius),
		spacing: clamp(src.spacing, 4, 12, defaultTokens.spacing),
		headingFont: font(src.headingFont, defaultTokens.headingFont),
		bodyFont: font(src.bodyFont, defaultTokens.bodyFont),
		customCss:
			typeof src.customCss === "string"
				? src.customCss.slice(0, MAX_CUSTOM_CSS)
				: "",
	};
};

// Where the current theme is kept: the user's account when signed in,
// otherwise just this browser.
export type ThemeSaveTarget = "account" | "browser";

export interface ThemeTokensContextValue {
	tokens: ThemeTokens;
	setTokens: (update: (prev: ThemeTokens) => ThemeTokens) => void;
	reset: () => void;
	savedTo: ThemeSaveTarget;
	saveError: string;
}

export const ThemeTokensContext = createContext<ThemeTokensContextValue>({
	tokens: defaultTokens,
	setTokens: () => {},
	reset: () => {},
	savedTo: "browser",
	saveError: "",
});

export const useThemeTokens = () => useContext(ThemeTokensContext);
