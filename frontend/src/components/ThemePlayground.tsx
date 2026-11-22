import React from "react";
import { IconButton, Tooltip } from "@mui/material";
import PaletteIcon from "@mui/icons-material/PaletteOutlined";

const ThemePanel = React.lazy(() => import("./ThemePanel"));

// App-bar button that opens the personal theme panel. The panel itself is
// loaded the first time it's opened.
export default function ThemePlayground() {
	const [open, setOpen] = React.useState(false);
	const [loaded, setLoaded] = React.useState(false);
	return (
		<>
			<Tooltip title="Customize theme">
				<IconButton
					size="small"
					aria-label="Customize theme"
					onClick={() => {
						setLoaded(true);
						setOpen((o) => !o);
					}}
					sx={{ color: open ? "primary.main" : "text.secondary" }}
				>
					<PaletteIcon fontSize="small" />
				</IconButton>
			</Tooltip>
			{loaded && (
				<React.Suspense fallback={null}>
					<ThemePanel open={open} onClose={() => setOpen(false)} />
				</React.Suspense>
			)}
		</>
	);
}
