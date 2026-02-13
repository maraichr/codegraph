import type { ImpactNode } from "../../api/types";

interface Props {
  totalAffected: number;
  directImpact: ImpactNode[];
  transitiveImpact: ImpactNode[];
}

const SEVERITY_COLORS: Record<string, { bg: string; text: string }> = {
  critical: { bg: "bg-red-100", text: "text-red-700" },
  high: { bg: "bg-orange-100", text: "text-orange-700" },
  medium: { bg: "bg-yellow-100", text: "text-yellow-700" },
  low: { bg: "bg-gray-100", text: "text-gray-600" },
};

export function ImpactSummary({
  totalAffected,
  directImpact,
  transitiveImpact,
}: Props) {
  const allNodes = [...directImpact, ...transitiveImpact];
  const counts: Record<string, number> = {};
  for (const node of allNodes) {
    counts[node.severity] = (counts[node.severity] || 0) + 1;
  }

  return (
    <div className="flex items-center gap-4 border-b border-gray-200 bg-white px-4 py-3">
      <div className="text-sm text-gray-700">
        <span className="font-semibold">{totalAffected}</span> affected symbol
        {totalAffected !== 1 ? "s" : ""}
      </div>
      <div className="h-4 w-px bg-gray-200" />
      <div className="text-xs text-gray-500">
        <span className="font-medium">{directImpact.length}</span> direct,{" "}
        <span className="font-medium">{transitiveImpact.length}</span>{" "}
        transitive
      </div>
      <div className="h-4 w-px bg-gray-200" />
      <div className="flex gap-2">
        {(["critical", "high", "medium", "low"] as const).map((sev) => {
          const count = counts[sev] || 0;
          if (count === 0) return null;
          const colors = SEVERITY_COLORS[sev];
          return (
            <span
              key={sev}
              className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${colors.bg} ${colors.text}`}
            >
              {count} {sev}
            </span>
          );
        })}
      </div>
    </div>
  );
}
