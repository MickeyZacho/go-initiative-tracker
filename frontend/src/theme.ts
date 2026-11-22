import { alpha, createTheme, type Theme } from "@mui/material/styles";

// The default fonts are loaded in index.html (Google Fonts); others the theme
// playground picks are loaded on demand by ThemeTokensProvider.

// The tweakable part of the look. The theme playground (ThemePlayground.tsx)
// edits a copy of these live and can copy the result back here as code.
export interface SchemeTokens {
	primary: string;
	primaryContrast: string;
	secondary: string;
	backgroundDefault: string;
	backgroundPaper: string;
	textPrimary: string;
	textSecondary: string;
}
export interface ThemeTokens {
	light: SchemeTokens;
	dark: SchemeTokens;
	borderRadius: number;
	spacing: number;
	headingFont: string;
	bodyFont: string;
	customCss: string;
}

export const defaultTokens: ThemeTokens = {
	light: {
		primary: "#b3372b",
		primaryContrast: "#ffffff",
		secondary: "#b07300",
		backgroundDefault: "#f2f2ec",
		backgroundPaper: "#fffaf0",
		textPrimary: "#645142",
		textSecondary: "#b3a89f",
	},
	dark: {
		primary: "#ec6a4c",
		primaryContrast: "#1a0e0a",
		secondary: "#e2b45c",
		backgroundDefault: "#13100e",
		backgroundPaper: "#1d1916",
		textPrimary: "#f1e9e1",
		textSecondary: "#a99c90",
	},
	borderRadius: 20,
	spacing: 6,
	headingFont: "Atkinson Hyperlegible",
	bodyFont: "IBM Plex Sans",
	customCss: "",
};

// Font stacks for the family names the tokens hold.
export const headingStack = (family: string) => `"${family}", Georgia, serif`;
export const bodyStack = (family: string) =>
	`"${family}", system-ui, -apple-system, "Segoe UI", Roboto, Arial, sans-serif`;

const schemePalette = (t: SchemeTokens) => ({
	primary: { main: t.primary, contrastText: t.primaryContrast },
	secondary: { main: t.secondary },
	background: { default: t.backgroundDefault, paper: t.backgroundPaper },
	text: { primary: t.textPrimary, secondary: t.textSecondary },
});

// One theme for the whole app, with light and dark color schemes. MUI follows
// the OS preference by default ("system"); the app-bar toggle lets a DM force
// dark mode at a dim table. Components should use palette tokens (sx colors
// like "primary.main", "text.secondary", "divider") rather than raw hex, so both
// schemes stay legible.
//
// The look is "warm ember on charcoal": a crimson/ember primary, a gold
// secondary (conditions), and warm neutrals instead of MUI's cold greys. Dark
// is the scheme that matters most — it's what's on screen at the table.
export const buildTheme = (tokens: ThemeTokens) => {
	const heading = headingStack(tokens.headingFont);
	return createTheme({
		cssVariables: { colorSchemeSelector: "class" },
		colorSchemes: {
			light: {
				palette: {
					...schemePalette(tokens.light),
					divider: "rgba(60, 40, 25, 0.12)",
				},
			},
			dark: {
				palette: {
					...schemePalette(tokens.dark),
					divider: "rgba(255, 235, 215, 0.10)",
				},
			},
		},
		shape: { borderRadius: tokens.borderRadius },
		spacing: tokens.spacing,
		typography: {
			fontFamily: bodyStack(tokens.bodyFont),
			h4: { fontFamily: heading, fontWeight: 700 },
			h5: { fontFamily: heading, fontWeight: 700, letterSpacing: "0.02em" },
			h6: { fontFamily: heading, fontWeight: 700 },
			button: { textTransform: "none", fontWeight: 600 },
			overline: { fontWeight: 600, letterSpacing: "0.08em" },
		},
		components: {
			MuiCssBaseline: {
				styleOverrides: {
					// Stat columns (HP, AC, initiative) line up when digits are equal width.
					body: { fontVariantNumeric: "tabular-nums" },
				},
			},
			MuiButton: { defaultProps: { disableElevation: true } },
			// MUI lightens dark-mode surfaces with a white gradient per elevation level;
			// that reads as grey haze on the warm palette, so use flat surfaces.
			MuiPaper: { styleOverrides: { root: { backgroundImage: "none" } } },
			MuiAppBar: {
				styleOverrides: {
					root: ({ theme }) => ({
						backgroundColor: theme.vars?.palette.background.paper,
						color: theme.vars?.palette.text.primary,
						borderBottom: `1px solid ${theme.vars?.palette.divider}`,
					}),
				},
			},
			MuiTableCell: {
				styleOverrides: {
					head: ({ theme }) => ({
						color: theme.vars?.palette.text.secondary,
						fontWeight: 600,
						fontSize: "0.75rem",
						textTransform: "uppercase",
						letterSpacing: "0.06em",
					}),
				},
			},
			MuiChip: { styleOverrides: { root: { fontWeight: 600 } } },
		},
	});
};

export const theme = buildTheme(defaultTokens);

type PaletteKey = "primary" | "secondary" | "error" | "warning" | "success" | "info";

// tint returns an sx color callback for a translucent wash of a palette color
// that tracks the active color scheme (via the CSS-variable channel), e.g.
// `sx={{ bgcolor: tint("primary", 0.12) }}`.
export const tint =
	(key: PaletteKey, opacity: number) =>
	(t: Theme): string =>
		t.vars
			? `rgba(${t.vars.palette[key].mainChannel} / ${opacity})`
			: alpha(t.palette[key].main, opacity);
