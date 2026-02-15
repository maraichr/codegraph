import { useState } from "react";
import type { ImpactNode } from "../../api/types";
import { Badge } from "../ui/badge";

interface Props {
  directImpact: ImpactNode[];
  transitiveImpact: ImpactNode[];
  viewMode: "tree" | "list";
}

const SEVERITY_VARIANT: Record<string, "destructive" | "warning" | "info" | "secondary"> = {
  critical: "destructive",
  high: "warning",
  medium: "info",
  low: "secondary",
};

function ImpactNodeRow({ node }: { node: ImpactNode }) {
  return (
    <div className="rounded-md border bg-card p-2">
      <div className="flex items-center gap-2">
        <Badge
          variant={SEVERITY_VARIANT[node.severity] ?? "secondary"}
          className="text-[10px] capitalize"
        >
          {node.severity}
        </Badge>
        <span className="text-sm font-medium">{node.symbol.name}</span>
        <Badge variant="outline" className="text-[10px]">
          {node.symbol.kind}
        </Badge>
      </div>
      <div className="mt-0.5 pl-4 text-xs text-muted-foreground">
        {node.symbol.qualified_name}
        <span className="ml-2">
          depth {node.depth} via {node.edge_type.replace(/_/g, " ")}
        </span>
      </div>
    </div>
  );
}

function TreeView({ directImpact, transitiveImpact }: Omit<Props, "viewMode">) {
  const [expandTransitive, setExpandTransitive] = useState(true);

  return (
    <div className="space-y-4 p-4">
      <div>
        <h4 className="mb-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          Direct Impact ({directImpact.length})
        </h4>
        <div className="space-y-1">
          {directImpact.map((node) => (
            <ImpactNodeRow key={node.symbol.id} node={node} />
          ))}
          {directImpact.length === 0 && (
            <p className="text-xs text-muted-foreground">No direct impact</p>
          )}
        </div>
      </div>

      {transitiveImpact.length > 0 && (
        <div>
          <button
            type="button"
            onClick={() => setExpandTransitive(!expandTransitive)}
            className="mb-2 flex items-center gap-1 text-xs font-semibold uppercase tracking-wider text-muted-foreground hover:text-foreground"
          >
            <span>{expandTransitive ? "\u25BC" : "\u25B6"}</span>
            Transitive Impact ({transitiveImpact.length})
          </button>
          {expandTransitive && (
            <div className="space-y-1 pl-4">
              {transitiveImpact.map((node) => (
                <ImpactNodeRow key={node.symbol.id} node={node} />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function ListView({ directImpact, transitiveImpact }: Omit<Props, "viewMode">) {
  const allNodes = [...directImpact, ...transitiveImpact].sort((a, b) => {
    const severityOrder = { critical: 0, high: 1, medium: 2, low: 3 };
    const sa = severityOrder[a.severity as keyof typeof severityOrder] ?? 4;
    const sb = severityOrder[b.severity as keyof typeof severityOrder] ?? 4;
    if (sa !== sb) return sa - sb;
    return a.depth - b.depth;
  });

  return (
    <div className="p-4">
      <table className="w-full text-left text-xs">
        <thead>
          <tr className="border-b text-muted-foreground">
            <th className="pb-2 font-medium">Symbol</th>
            <th className="pb-2 font-medium">Kind</th>
            <th className="pb-2 font-medium">Severity</th>
            <th className="pb-2 font-medium">Depth</th>
            <th className="pb-2 font-medium">Edge Type</th>
          </tr>
        </thead>
        <tbody className="divide-y">
          {allNodes.map((node) => (
            <tr key={node.symbol.id} className="hover:bg-muted/30">
              <td className="py-1.5">
                <div className="font-medium">{node.symbol.name}</div>
                <div className="text-muted-foreground">{node.symbol.qualified_name}</div>
              </td>
              <td className="py-1.5">
                <Badge variant="outline" className="text-[10px]">
                  {node.symbol.kind}
                </Badge>
              </td>
              <td className="py-1.5">
                <Badge
                  variant={SEVERITY_VARIANT[node.severity] ?? "secondary"}
                  className="text-[10px] capitalize"
                >
                  {node.severity}
                </Badge>
              </td>
              <td className="py-1.5 text-muted-foreground">{node.depth}</td>
              <td className="py-1.5 text-muted-foreground">{node.edge_type.replace(/_/g, " ")}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function ImpactTree({ directImpact, transitiveImpact, viewMode }: Props) {
  if (viewMode === "list") {
    return <ListView directImpact={directImpact} transitiveImpact={transitiveImpact} />;
  }
  return <TreeView directImpact={directImpact} transitiveImpact={transitiveImpact} />;
}
