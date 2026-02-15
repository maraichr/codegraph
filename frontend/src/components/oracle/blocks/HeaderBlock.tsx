import type { OracleHeaderData } from "../../../api/types";

interface Props {
  data: unknown;
}

export function HeaderBlock({ data }: Props) {
  const d = data as OracleHeaderData;
  // Strip markdown bold markers for clean rendering
  const text = d.text?.replace(/\*\*/g, "") || "";

  return (
    <h3 className="text-sm font-semibold text-foreground border-b border-border/50 pb-1.5">
      {text}
    </h3>
  );
}
