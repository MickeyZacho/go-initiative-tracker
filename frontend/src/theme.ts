import { alpha, createTheme, type Theme } from "@mui/material/styles";

// Fonts are loaded in index.html (Google Fonts). Cinzel is a display face for
// the brand and page headings only; everything you read or type is Inter.
export const headingFont = '"Cinzel", Georgia, serif';
const bodyFont =
	'"Inter", system-ui, -apple-system, "Segoe UI", Roboto, Arial, sans-serif';

// One theme for the whole app, with light and dark color schemes. MUI follows
// the OS preference by default ("system"); the app-bar toggle lets a DM force
// dark mode at a dim table. Components should use palette tokens (sx colors
// like "primary.main", "text.secondary", "divider") rather than raw hex, so both
// schemes stay legible.
//
// The look is "warm ember on charcoal": a crimson/ember primary, a gold
// secondary (conditions), and warm neutrals instead of MUI's cold greys. Dark
// is the scheme that matters most — it's what's on screen at the table.
export const theme = createTheme({
	cssVariables: { colorSchemeSelector: "class" },
	colorSchemes: {
		light: {
			palette: {
				primary: { main: "#b3372b", contrastText: "#ffffff" },
				secondary: { main: "#8a5a00" },
				background: { default: "#f5f1ea", paper: "#fffdf9" },
				text: { primary: "#231c17", secondary: "#6b5f55" },
				divider: "rgba(60, 40, 25, 0.12)",
			},
		},
		dark: {
			palette: {
				primary: { main: "#ec6a4c", contrastText: "#1a0e0a" },
				secondary: { main: "#e2b45c" },
				background: { default: "#13100e", paper: "#1d1916" },
				text: { primary: "#f1e9e1", secondary: "#a99c90" },
				divider: "rgba(255, 235, 215, 0.10)",
			},
		},
	},
	shape: { borderRadius: 8 },
	typography: {
		fontFamily: bodyFont,
		h4: { fontFamily: headingFont, fontWeight: 700 },
		h5: { fontFamily: headingFont, fontWeight: 700, letterSpacing: "0.02em" },
		h6: { fontFamily: headingFont, fontWeight: 700 },
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
