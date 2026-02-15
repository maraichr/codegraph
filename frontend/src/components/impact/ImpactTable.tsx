import { useState } from "react";
import type { ImpactNode } from "../../api/types";
import { Badge } from "../ui/badge";

interface Props {
  nodes: ImpactNode[];
}

type SortField = "name" | "severity" | "depth" | "kind";

const SEVERITY_ORDER: Record<string, number> = { critical: 0, high: 1, medium: 2, low: 3 };
const SEVERITY_VARIANT: Record<string, "destructive" | "warning" | "info" | "secondary"> = {
  critical: "destructive",
  high: "warning",
  medium: "info",
  low: "secondary",
};

export function ImpactTable({ nodes }: Props) {
  const [sortField, setSortField] = useState<SortField>("severity");
  const [sortAsc, setSortAsc] = useState(true);

  const toggleSort = (field: SortField) => {
    if (sortField === field) {
      setSortAsc(!sortAsc);
    } else {
      setSortField(field);
      setSortAsc(true);
    }
  };

  const sorted = [...nodes].sort((a, b) => {
    let cmp = 0;
    switch (sortField) {
      case "name":
        cmp = a.symbol.name.localeCompare(b.symbol.name);
        break;
      case "severity":
        cmp = (SEVERITY_ORDER[a.severity] ?? 4) - (SEVERITY_ORDER[b.severity] ?? 4);
        break;
      case "depth":
        cmp = a.depth - b.depth;
        break;
      case "kind":
        cmp = a.symbol.kind.localeCompare(b.symbol.kind);
        break;
    }
    return sortAsc ? cmp : -cmp;
  });

  const headerClass =
    "cursor-pointer select-none px-3 py-2 text-left text-xs font-medium text-muted-foreground hover:text-foreground";

  return (
    <div className="overflow-auto rounded-md border">
      <table className="w-full text-sm">
        <thead className="border-b bg-muted/50">
          <tr>
            <th className={headerClass} onClick={() => toggleSort("name")}>
              Symbol {sortField === "name" && (sortAsc ? "\u2191" : "\u2193")}
            </th>
            <th className={headerClass} onClick={() => toggleSort("kind")}>
              Kind {sortField === "kind" && (sortAsc ? "\u2191" : "\u2193")}
            </th>
            <th className={headerClass} onClick={() => toggleSort("severity")}>
              Severity {sortField === "severity" && (sortAsc ? "\u2191" : "\u2193")}
            </th>
            <th className={headerClass} onClick={() => toggleSort("depth")}>
              Depth {sortField === "depth" && (sortAsc ? "\u2191" : "\u2193")}
            </th>
            <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground">
              Edge Type
            </th>
          </tr>
        </thead>
        <tbody className="divide-y">
          {sorted.map((node) => (
            <tr key={node.symbol.id} className="hover:bg-muted/30">
              <td className="px-3 py-2">
                <div className="font-medium">{node.symbol.name}</div>
                <div className="text-xs text-muted-foreground">{node.symbol.qualified_name}</div>
              </td>
              <td className="px-3 py-2">
                <Badge variant="outline" className="text-xs">
                  {node.symbol.kind}
                </Badge>
              </td>
              <td className="px-3 py-2">
                <Badge
                  variant={SEVERITY_VARIANT[node.severity] ?? "secondary"}
                  className="text-xs capitalize"
                >
                  {node.severity}
                </Badge>
              </td>
              <td className="px-3 py-2 text-muted-foreground">{node.depth}</td>
              <td className="px-3 py-2 text-muted-foreground">
                {node.edge_type.replace(/_/g, " ")}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
