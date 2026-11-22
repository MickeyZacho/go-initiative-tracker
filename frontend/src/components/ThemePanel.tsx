import React from "react";
import {
	Box,
	Button,
	Divider,
	Drawer,
	IconButton,
	MenuItem,
	Slider,
	Stack,
	TextField,
	Typography,
} from "@mui/material";
import { useColorScheme } from "@mui/material/styles";
import CloseIcon from "@mui/icons-material/Close";
import type { SchemeTokens, ThemeTokens } from "../theme";
import {
	MAX_CUSTOM_CSS,
	PLAYGROUND_FONTS,
	useThemeTokens,
} from "../lib/themeTokens";

// "Copy as code" is for changing the app default in theme.ts, so only devs
// see it.
const SHOW_COPY_AS_CODE = import.meta.env.DEV;

const COLOR_FIELDS: { key: keyof SchemeTokens; label: string }[] = [
	{ key: "primary", label: "Primary" },
	{ key: "primaryContrast", label: "Text on primary" },
	{ key: "secondary", label: "Secondary" },
	{ key: "backgroundDefault", label: "Page background" },
	{ key: "backgroundPaper", label: "Card background" },
	{ key: "textPrimary", label: "Text" },
	{ key: "textSecondary", label: "Muted text" },
];

const HEX = /^#[0-9a-f]{6}$/i;

// One color token: a native picker plus a hex box. The box only commits a
// complete #rrggbb (MUI throws on partial colors); it re-syncs from the value
// whenever the picker changes it.
function ColorRow({
	label,
	value,
	onChange,
}: {
	label: string;
	value: string;
	onChange: (v: string) => void;
}) {
	return (
		<Stack direction="row" spacing={1} alignItems="center">
			<Box
				component="input"
				type="color"
				value={value}
				onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
					onChange(e.target.value)
				}
				aria-label={label}
				sx={{
					width: 36,
					height: 30,
					p: 0,
					border: 1,
					borderColor: "divider",
					borderRadius: 1,
					bgcolor: "transparent",
					cursor: "pointer",
					flexShrink: 0,
				}}
			/>
			<Typography variant="body2" sx={{ flex: 1 }}>
				{label}
			</Typography>
			<TextField
				key={value}
				size="small"
				defaultValue={value}
				onChange={(e) => {
					if (HEX.test(e.target.value)) onChange(e.target.value);
				}}
				slotProps={{
					htmlInput: { "aria-label": `${label} hex`, spellCheck: false },
				}}
				sx={{ width: 96, "& input": { fontFamily: "monospace", py: 0.5 } }}
			/>
		</Stack>
	);
}

const asCode = (tokens: ThemeTokens) =>
	`export const defaultTokens: ThemeTokens = ${JSON.stringify(tokens, null, "\t")};\n`;

// Side panel where anyone can restyle the app for themselves: colours,
// shapes, fonts and custom CSS, applied live. Saved to their account when
// signed in (see ThemeTokensProvider), otherwise in this browser. In dev,
// "Copy as code" gives a defaultTokens block for theme.ts.
// Lazy-loaded by ThemePlayground (the app-bar button) so the panel's code
// isn't in the main bundle.
export default function ThemePanel({
	open,
	onClose,
}: {
	open: boolean;
	onClose: () => void;
}) {
	const [copied, setCopied] = React.useState(false);
	const { tokens, setTokens, reset, savedTo, saveError } = useThemeTokens();
	const { colorScheme } = useColorScheme();
	const scheme = colorScheme === "dark" ? "dark" : "light";

	const set = <K extends keyof ThemeTokens>(key: K, value: ThemeTokens[K]) =>
		setTokens((prev) => ({ ...prev, [key]: value }));
	const setColor = (key: keyof SchemeTokens, value: string) =>
		setTokens((prev) => ({
			...prev,
			[scheme]: { ...prev[scheme], [key]: value },
		}));

	const copy = async () => {
		try {
			await navigator.clipboard.writeText(asCode(tokens));
			setCopied(true);
			setTimeout(() => setCopied(false), 1500);
		} catch {
			setCopied(false);
		}
	};

	return (
		<>
			{/* Persistent, not modal: no backdrop, so the app stays visible and
			    usable while you tweak. */}
			<Drawer
				anchor="right"
				variant="persistent"
				open={open}
				sx={{ zIndex: (t) => t.zIndex.appBar + 1 }}
				slotProps={{
					paper: {
						sx: {
							width: { xs: "100%", sm: 340 },
							p: 2,
							borderLeft: 1,
							borderColor: "divider",
						},
					},
				}}
			>
				<Stack spacing={2}>
					<Stack direction="row" alignItems="center">
						<Typography variant="h6" sx={{ flex: 1 }}>
							Your theme
						</Typography>
						<IconButton
							size="small"
							aria-label="Close theme panel"
							onClick={onClose}
						>
							<CloseIcon fontSize="small" />
						</IconButton>
					</Stack>

					<Typography variant="overline" color="text.secondary">
						Colours · {scheme} scheme
					</Typography>
					<Stack spacing={1} sx={{ mt: "0 !important" }}>
						{COLOR_FIELDS.map((f) => (
							<ColorRow
								key={`${scheme}-${f.key}`}
								label={f.label}
								value={tokens[scheme][f.key]}
								onChange={(v) => setColor(f.key, v)}
							/>
						))}
					</Stack>
					<Typography variant="caption" color="text.secondary">
						Switch light/dark with the toggle to edit the other scheme.
					</Typography>

					<Divider />
					<Typography variant="overline" color="text.secondary">
						Shape
					</Typography>
					<Box sx={{ mt: "0 !important", px: 1 }}>
						<Typography variant="body2">
							Corner radius: {tokens.borderRadius}px
						</Typography>
						<Slider
							size="small"
							min={0}
							max={24}
							value={tokens.borderRadius}
							onChange={(_, v) => set("borderRadius", v as number)}
						/>
						<Typography variant="body2">
							Spacing unit: {tokens.spacing}px
						</Typography>
						<Slider
							size="small"
							min={4}
							max={12}
							value={tokens.spacing}
							onChange={(_, v) => set("spacing", v as number)}
						/>
					</Box>

					<Divider />
					<Typography variant="overline" color="text.secondary">
						Fonts
					</Typography>
					{(
						[
							["headingFont", "Headings"],
							["bodyFont", "Body"],
						] as const
					).map(([key, label]) => (
						<TextField
							key={key}
							select
							size="small"
							label={label}
							value={tokens[key]}
							onChange={(e) => set(key, e.target.value)}
						>
							{PLAYGROUND_FONTS.map((f) => (
								<MenuItem
									key={f.family}
									value={f.family}
									sx={{ fontFamily: `"${f.family}"` }}
								>
									{f.family}
								</MenuItem>
							))}
						</TextField>
					))}

					<Divider />
					<Typography variant="overline" color="text.secondary">
						Custom CSS
					</Typography>
					<TextField
						multiline
						minRows={6}
						value={tokens.customCss}
						onChange={(e) =>
							set("customCss", e.target.value.slice(0, MAX_CUSTOM_CSS))
						}
						placeholder={".character-row {\n  border-width: 2px;\n}"}
						slotProps={{ htmlInput: { spellCheck: false } }}
						sx={{
							mt: "0 !important",
							"& textarea": { fontFamily: "monospace", fontSize: "0.8rem" },
						}}
					/>

					<Typography
						variant="caption"
						color={saveError ? "error" : "text.secondary"}
					>
						{saveError ||
							(savedTo === "account"
								? "Changes save to your account automatically."
								: "Saved in this browser only. Log in to keep your theme on every device.")}
					</Typography>
					<Stack direction="row" spacing={1}>
						{SHOW_COPY_AS_CODE && (
							<Button variant="outlined" onClick={copy} sx={{ flex: 1 }}>
								{copied ? "Copied!" : "Copy as code"}
							</Button>
						)}
						<Button variant="outlined" onClick={reset} sx={{ flex: 1 }}>
							Reset to default
						</Button>
					</Stack>
				</Stack>
			</Drawer>
		</>
	);
}
