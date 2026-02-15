import type { OracleTextData } from "../../../api/types";

interface Props {
  data: unknown;
}

export function TextBlock({ data }: Props) {
  const d = data as OracleTextData;
  if (!d.content) return null;

  // Simple markdown-like rendering: bold and code spans
  const rendered = d.content
    .replace(/\*\*(.*?)\*\*/g, '<strong class="font-semibold text-foreground">$1</strong>')
    .replace(/`(.*?)`/g, '<code class="rounded bg-muted px-1 py-0.5 text-[11px] font-mono">$1</code>');

  return (
    <div
      className="text-xs text-muted-foreground leading-relaxed"
      dangerouslySetInnerHTML={{ __html: rendered }}
    />
  );
}
