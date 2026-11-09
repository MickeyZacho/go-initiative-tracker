import { Box, Typography } from "@mui/material";

interface DeathSavesProps {
	name: string;
	successes: number;
	failures: number;
	onChange: (successes: number, failures: number) => void;
	disabled?: boolean;
}

// Three success pips and three failure pips for a PC at 0 HP. Clicking pip n
// sets the count to n; clicking the last filled pip again clears it, so a
// mis-tap is one click to undo.
export function DeathSaves({
	name,
	successes,
	failures,
	onChange,
	disabled,
}: DeathSavesProps) {
	const status =
		failures >= 3 ? "Dead" : successes >= 3 ? "Stable" : "Making death saves";

	const pips = (
		kind: "success" | "failure",
		count: number,
		set: (n: number) => void,
	) =>
		[1, 2, 3].map((n) => {
			const filled = n <= count;
			const color = kind === "success" ? "success.main" : "error.main";
			return (
				<Box
					key={n}
					component="button"
					type="button"
					disabled={disabled}
					aria-label={`${name} death save ${kind} ${n}${filled ? " (marked)" : ""}`}
					aria-pressed={filled}
					onClick={() => set(filled && n === count ? n - 1 : n)}
					sx={{
						width: 18,
						height: 18,
						p: 0,
						borderRadius: "50%",
						border: 2,
						borderColor: color,
						bgcolor: filled ? color : "transparent",
						cursor: disabled ? "default" : "pointer",
					}}
				/>
			);
		});

	return (
		<Box
			sx={{
				display: "flex",
				alignItems: "center",
				flexWrap: "wrap",
				gap: 1,
				mt: 1,
			}}
		>
			<Typography variant="caption" color="primary" sx={{ fontWeight: 600 }}>
				Death saves:
			</Typography>
			<Box sx={{ display: "flex", gap: 0.5, alignItems: "center" }}>
				<Typography variant="caption" color="success.main">
					✓
				</Typography>
				{pips("success", successes, (n) => onChange(n, failures))}
			</Box>
			<Box sx={{ display: "flex", gap: 0.5, alignItems: "center" }}>
				<Typography variant="caption" color="error.main">
					✗
				</Typography>
				{pips("failure", failures, (n) => onChange(successes, n))}
			</Box>
			<Typography
				variant="caption"
				sx={{ fontWeight: 700 }}
				color={
					failures >= 3
						? "error.main"
						: successes >= 3
							? "success.main"
							: "text.secondary"
				}
			>
				{status}
			</Typography>
		</Box>
	);
}
