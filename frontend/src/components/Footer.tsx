import { useEffect, useState } from "react";
import { Box, Container, Link, Stack, SvgIcon, Typography } from "@mui/material";
import type { SvgIconProps } from "@mui/material";
import { apiGet } from "../lib/http";

const GITHUB_URL = "https://github.com/MickeyZacho/go-initiative-tracker";

// Inlined so we don't pull in @mui/icons-material just for one glyph. Path is
// the standard GitHub mark (MIT-licensed, from the Simple Icons / MUI set).
function GitHubIcon(props: SvgIconProps) {
	return (
		<SvgIcon viewBox="0 0 24 24" {...props}>
			<path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12" />
		</SvgIcon>
	);
}

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
					<Link
						href={GITHUB_URL}
						target="_blank"
						rel="noopener noreferrer"
						color="text.secondary"
						aria-label="Initiative Tracker on GitHub"
						sx={{ display: "inline-flex", "&:hover": { color: "text.primary" } }}
					>
						<GitHubIcon sx={{ fontSize: 18 }} />
					</Link>
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
