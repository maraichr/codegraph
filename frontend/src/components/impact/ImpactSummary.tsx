import type { ImpactNode } from "../../api/types";
import { Badge } from "../ui/badge";
import { Separator } from "../ui/separator";

interface Props {
  totalAffected: number;
  directImpact: ImpactNode[];
  transitiveImpact: ImpactNode[];
}

const SEVERITY_VARIANT: Record<string, "destructive" | "warning" | "info" | "secondary"> = {
  critical: "destructive",
  high: "warning",
  medium: "info",
  low: "secondary",
};

export function ImpactSummary({ totalAffected, directImpact, transitiveImpact }: Props) {
  const allNodes = [...directImpact, ...transitiveImpact];
  const counts: Record<string, number> = {};
  for (const node of allNodes) {
    counts[node.severity] = (counts[node.severity] || 0) + 1;
  }

  return (
    <div className="flex items-center gap-4 border-b bg-card px-4 py-3">
      <div className="text-sm">
        <span className="font-semibold">{totalAffected}</span> affected symbol
        {totalAffected !== 1 ? "s" : ""}
      </div>
      <Separator orientation="vertical" className="h-4" />
      <div className="text-xs text-muted-foreground">
        <span className="font-medium">{directImpact.length}</span> direct,{" "}
        <span className="font-medium">{transitiveImpact.length}</span> transitive
      </div>
      <Separator orientation="vertical" className="h-4" />
      <div className="flex gap-2">
        {(["critical", "high", "medium", "low"] as const).map((sev) => {
          const count = counts[sev] || 0;
          if (count === 0) return null;
          return (
            <Badge key={sev} variant={SEVERITY_VARIANT[sev]} className="text-[10px] capitalize">
              {count} {sev}
            </Badge>
          );
        })}
      </div>
    </div>
  );
}
