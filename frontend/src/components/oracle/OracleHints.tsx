import type { OracleHint } from "../../api/types";

interface Props {
  hints: OracleHint[];
  onClick: (question: string) => void;
}

export function OracleHints({ hints, onClick }: Props) {
  return (
    <div className="flex flex-wrap gap-1.5 pt-1">
      {hints.map((hint, i) => (
        <button
          key={i}
          onClick={() => onClick(hint.question)}
          className="rounded-full border border-border bg-muted/30 px-2.5 py-1 text-[11px] text-muted-foreground transition-colors hover:bg-muted/60 hover:text-foreground hover:border-primary/30"
          title={hint.question}
        >
          {hint.label}
        </button>
      ))}
    </div>
  );
}
