import { Badge } from "../../ui/badge";
import type { OracleSymbolItem, OracleSymbolListData } from "../../../api/types";

const KIND_VARIANT: Record<string, "default" | "info" | "success" | "warning" | "secondary"> = {
  table: "info",
  view: "secondary",
  procedure: "success",
  function: "warning",
  class: "info",
  column: "secondary",
  interface: "secondary",
  method: "success",
};

interface Props {
  data: unknown;
}

export function SymbolListBlock({ data }: Props) {
  const d = data as OracleSymbolListData;
  if (!d.symbols?.length) return null;

  return (
    <div className="space-y-1.5">
      {d.symbols.map((sym: OracleSymbolItem) => (
        <div
          key={sym.id}
          className="flex items-center gap-2 rounded-md border border-border/50 bg-muted/20 px-2.5 py-2 text-xs transition-colors hover:bg-muted/40"
        >
          <Badge variant={KIND_VARIANT[sym.kind] || "secondary"} className="text-[10px] shrink-0">
            {sym.kind}
          </Badge>
          <div className="min-w-0 flex-1">
            <span className="font-medium text-foreground">{sym.name}</span>
            {sym.qualified_name !== sym.name && (
              <span className="ml-1.5 text-muted-foreground truncate block text-[10px]">
                {sym.qualified_name}
              </span>
            )}
          </div>
          <span className="shrink-0 text-[10px] text-muted-foreground">{sym.language}</span>
        </div>
      ))}
    </div>
  );
}
