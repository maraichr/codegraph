import { Badge } from "../ui/badge";
import { Button } from "../ui/button";

interface Props {
  severityFilter: string[];
  onSeverityChange: (severity: string[]) => void;
  maxDepth: number;
  onMaxDepthChange: (depth: number) => void;
}

const SEVERITIES = ["critical", "high", "medium", "low"];

export function ImpactFilters({
  severityFilter,
  onSeverityChange,
  maxDepth,
  onMaxDepthChange,
}: Props) {
  const toggleSeverity = (sev: string) => {
    if (severityFilter.includes(sev)) {
      onSeverityChange(severityFilter.filter((s) => s !== sev));
    } else {
      onSeverityChange([...severityFilter, sev]);
    }
  };

  return (
    <div className="flex items-center gap-4 rounded-md border bg-card p-3">
      <div className="flex items-center gap-2">
        <span className="text-xs font-medium text-muted-foreground">Severity:</span>
        {SEVERITIES.map((sev) => (
          <Badge
            key={sev}
            variant={
              severityFilter.includes(sev) || severityFilter.length === 0 ? "default" : "outline"
            }
            className="cursor-pointer text-xs capitalize"
            onClick={() => toggleSeverity(sev)}
          >
            {sev}
          </Badge>
        ))}
      </div>
      <div className="flex items-center gap-2">
        <span className="text-xs font-medium text-muted-foreground">Depth:</span>
        {[3, 5, 7, 10].map((d) => (
          <Button
            key={d}
            variant={maxDepth === d ? "default" : "ghost"}
            size="sm"
            onClick={() => onMaxDepthChange(d)}
          >
            {d}
          </Button>
        ))}
      </div>
    </div>
  );
}
