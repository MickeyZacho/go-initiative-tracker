import { IconButton, Tooltip } from "@mui/material";
import { useColorScheme } from "@mui/material/styles";
import BrightnessAutoIcon from "@mui/icons-material/BrightnessAuto";
import DarkModeIcon from "@mui/icons-material/DarkModeOutlined";
import LightModeIcon from "@mui/icons-material/LightModeOutlined";

const NEXT = { system: "light", light: "dark", dark: "system" } as const;
const LABEL = { system: "Auto", light: "Light", dark: "Dark" } as const;
const ICON = {
	system: BrightnessAutoIcon,
	light: LightModeIcon,
	dark: DarkModeIcon,
} as const;

// Cycles system → light → dark. MUI persists the choice in localStorage.
export default function ColorModeToggle() {
	const { mode, setMode } = useColorScheme();
	if (!mode) return null; // not resolved yet (first render)
	const Icon = ICON[mode];
	return (
		<Tooltip title={`Theme: ${LABEL[mode]} (click to change)`}>
			<IconButton
				size="small"
				aria-label={`Theme: ${LABEL[mode]}`}
				onClick={() => setMode(NEXT[mode])}
				sx={{ color: "text.secondary" }}
			>
				<Icon fontSize="small" />
			</IconButton>
		</Tooltip>
	);
}
