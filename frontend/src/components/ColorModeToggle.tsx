import { IconButton, Tooltip } from "@mui/material";
import { useColorScheme } from "@mui/material/styles";

const NEXT = { system: "light", light: "dark", dark: "system" } as const;
const LABEL = { system: "Auto", light: "Light", dark: "Dark" } as const;
const GLYPH = { system: "◐", light: "☀", dark: "☾" } as const;

// Cycles system → light → dark. MUI persists the choice in localStorage.
export default function ColorModeToggle() {
	const { mode, setMode } = useColorScheme();
	if (!mode) return null; // not resolved yet (first render)
	return (
		<Tooltip title={`Theme: ${LABEL[mode]} (click to change)`}>
			<IconButton
				size="small"
				aria-label={`Theme: ${LABEL[mode]}`}
				onClick={() => setMode(NEXT[mode])}
				sx={{ fontSize: "1.1rem", width: 32, height: 32 }}
			>
				{GLYPH[mode]}
			</IconButton>
		</Tooltip>
	);
}
