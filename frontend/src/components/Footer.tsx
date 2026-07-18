import { useEffect, useState } from "react";
import { Box, Container, Link, Stack, Typography } from "@mui/material";
import { apiGet } from "../lib/http";

const GITHUB_URL = "https://github.com/MickeyZacho/go-initiative-tracker";

// Baked into the bundle at build time (see vite.config.ts / version.d.ts).
const FRONTEND_VERSION = __APP_VERSION__;

/** Link a commit SHA to its GitHub page; bare hashes only, otherwise the repo. */
function commitUrl(sha: string): string {
	return /^[0-9a-f]{7,40}$/i.test(sha)
		? `${GITHUB_URL}/commit/${sha}`
		: GITHUB_URL;
}

function VersionLink({ label, sha }: { label: string; sha: string }) {
	return (
		<Typography variant="caption" color="text.secondary">
			{label}{" "}
			<Link
				href={commitUrl(sha)}
				target="_blank"
				rel="noopener noreferrer"
				color="inherit"
				underline="hover"
				sx={{ fontFamily: "monospace" }}
			>
				{sha}
			</Link>
		</Typography>
	);
}

/**
 * Site footer showing the deployed build. Frontend version is baked in at build
 * time; backend version is fetched from /api/version so it reflects the actual
 * running server (the two are deployed independently and can differ).
 */
export default function Footer() {
	const [backendVersion, setBackendVersion] = useState("…");

	useEffect(() => {
		let active = true;
		apiGet<{ version: string }>("/version")
			.then((data) => {
				if (active) setBackendVersion(data.version || "unknown");
			})
			.catch(() => {
				if (active) setBackendVersion("unavailable");
			});
		return () => {
			active = false;
		};
	}, []);

	return (
		<Box component="footer" sx={{ py: 2, px: 2 }}>
			<Container maxWidth="md" disableGutters>
				<Stack
					direction="row"
					spacing={1.5}
					justifyContent="center"
					alignItems="center"
					flexWrap="wrap"
					sx={{ rowGap: 0.5 }}
				>
					<Typography variant="caption" color="text.secondary">
						<Link
							href={GITHUB_URL}
							target="_blank"
							rel="noopener noreferrer"
							color="inherit"
							underline="hover"
						>
							Initiative Tracker on GitHub
						</Link>
					</Typography>
					<Typography variant="caption" color="text.disabled">
						·
					</Typography>
					<VersionLink label="frontend" sha={FRONTEND_VERSION} />
					<Typography variant="caption" color="text.disabled">
						·
					</Typography>
					<VersionLink label="backend" sha={backendVersion} />
				</Stack>
			</Container>
		</Box>
	);
}
