import { useState } from "react";
import type { ImpactNode } from "../../api/types";

interface Props {
  directImpact: ImpactNode[];
  transitiveImpact: ImpactNode[];
  viewMode: "tree" | "list";
}

const SEVERITY_STYLES: Record<
  string,
  { dot: string; bg: string; text: string }
> = {
  critical: {
    dot: "bg-red-500",
    bg: "bg-red-50 border-red-200",
    text: "text-red-700",
  },
  high: {
    dot: "bg-orange-500",
    bg: "bg-orange-50 border-orange-200",
    text: "text-orange-700",
  },
  medium: {
    dot: "bg-yellow-400",
    bg: "bg-yellow-50 border-yellow-200",
    text: "text-yellow-700",
  },
  low: {
    dot: "bg-gray-300",
    bg: "bg-gray-50 border-gray-200",
    text: "text-gray-600",
  },
};

function ImpactNodeRow({ node }: { node: ImpactNode }) {
  const style = SEVERITY_STYLES[node.severity] || SEVERITY_STYLES.low;

  return (
    <div className={`rounded border p-2 ${style.bg}`}>
      <div className="flex items-center gap-2">
        <span className={`inline-block h-2 w-2 rounded-full ${style.dot}`} />
        <span className="text-sm font-medium text-gray-900">
          {node.symbol.name}
        </span>
        <span className="rounded bg-white/60 px-1 text-[10px] text-gray-500">
          {node.symbol.kind}
        </span>
        <span className={`ml-auto text-[10px] font-medium ${style.text}`}>
          {node.severity}
        </span>
      </div>
      <div className="mt-0.5 pl-4 text-xs text-gray-500">
        {node.symbol.qualified_name}
        <span className="ml-2 text-gray-400">
          depth {node.depth} via {node.edge_type.replace(/_/g, " ")}
        </span>
      </div>
    </div>
  );
}

function TreeView({
  directImpact,
  transitiveImpact,
}: Omit<Props, "viewMode">) {
  const [expandTransitive, setExpandTransitive] = useState(true);

  return (
    <div className="space-y-4 p-4">
      <div>
        <h4 className="mb-2 text-xs font-semibold uppercase tracking-wider text-gray-500">
          Direct Impact ({directImpact.length})
        </h4>
        <div className="space-y-1">
          {directImpact.map((node) => (
            <ImpactNodeRow key={node.symbol.id} node={node} />
          ))}
          {directImpact.length === 0 && (
            <p className="text-xs text-gray-400">No direct impact</p>
          )}
        </div>
      </div>

      {transitiveImpact.length > 0 && (
        <div>
          <button
            type="button"
            onClick={() => setExpandTransitive(!expandTransitive)}
            className="mb-2 flex items-center gap-1 text-xs font-semibold uppercase tracking-wider text-gray-500 hover:text-gray-700"
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

function ListView({
  directImpact,
  transitiveImpact,
}: Omit<Props, "viewMode">) {
  const allNodes = [...directImpact, ...transitiveImpact].sort((a, b) => {
    const severityOrder = { critical: 0, high: 1, medium: 2, low: 3 };
    const sa =
      severityOrder[a.severity as keyof typeof severityOrder] ?? 4;
    const sb =
      severityOrder[b.severity as keyof typeof severityOrder] ?? 4;
    if (sa !== sb) return sa - sb;
    return a.depth - b.depth;
  });

  return (
    <div className="p-4">
      <table className="w-full text-left text-xs">
        <thead>
          <tr className="border-b border-gray-200 text-gray-500">
            <th className="pb-2 font-medium">Symbol</th>
            <th className="pb-2 font-medium">Kind</th>
            <th className="pb-2 font-medium">Severity</th>
            <th className="pb-2 font-medium">Depth</th>
            <th className="pb-2 font-medium">Edge Type</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-100">
          {allNodes.map((node) => {
            const style =
              SEVERITY_STYLES[node.severity] || SEVERITY_STYLES.low;
            return (
              <tr key={node.symbol.id}>
                <td className="py-1.5">
                  <div className="font-medium text-gray-900">
                    {node.symbol.name}
                  </div>
                  <div className="text-gray-400">
                    {node.symbol.qualified_name}
                  </div>
                </td>
                <td className="py-1.5 text-gray-600">{node.symbol.kind}</td>
                <td className="py-1.5">
                  <span
                    className={`inline-flex items-center gap-1 rounded-full px-1.5 py-0.5 text-[10px] font-medium ${style.bg} ${style.text}`}
                  >
                    <span
                      className={`inline-block h-1.5 w-1.5 rounded-full ${style.dot}`}
                    />
                    {node.severity}
                  </span>
                </td>
                <td className="py-1.5 text-gray-600">{node.depth}</td>
                <td className="py-1.5 text-gray-600">
                  {node.edge_type.replace(/_/g, " ")}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

export function ImpactTree({ directImpact, transitiveImpact, viewMode }: Props) {
  if (viewMode === "list") {
    return (
      <ListView
        directImpact={directImpact}
        transitiveImpact={transitiveImpact}
      />
    );
  }
  return (
    <TreeView
      directImpact={directImpact}
      transitiveImpact={transitiveImpact}
    />
  );
}
