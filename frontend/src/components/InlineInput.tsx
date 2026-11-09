import { InputBase, type InputBaseProps } from "@mui/material";

// An always-editable input that reads as plain text: no underline or box until
// the pointer is over it or it has focus. Used for the library lists (Characters,
// NPCs) so a roster looks like a roster rather than a stack of form fields.
export function InlineInput({ sx, ...props }: InputBaseProps) {
	return (
		<InputBase
			{...props}
			sx={[
				{
					px: 0.75,
					borderRadius: 1,
					border: "1px solid transparent",
					transition: "border-color 0.15s, background-color 0.15s",
					"&:hover": { bgcolor: "action.hover" },
					"&.Mui-focused": {
						borderColor: "primary.main",
						bgcolor: "background.default",
					},
					"& input": { py: 0.5 },
				},
				...(Array.isArray(sx) ? sx : [sx]),
			]}
		/>
	);
}
