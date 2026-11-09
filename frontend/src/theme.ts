import { alpha, createTheme, type Theme } from "@mui/material/styles";

// One theme for the whole app, with light and dark color schemes. MUI follows
// the OS preference by default ("system"); the app-bar toggle lets a DM force
// dark mode at a dim table. Components should use palette tokens (sx colors
// like "primary.main", "text.secondary", "divider") rather than raw hex, so both
// schemes stay legible.
export const theme = createTheme({
	cssVariables: { colorSchemeSelector: "class" },
	colorSchemes: {
		light: true,
		dark: {
			palette: {
				background: { default: "#121212", paper: "#1c1c1f" },
			},
		},
	},
	shape: { borderRadius: 8 },
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
