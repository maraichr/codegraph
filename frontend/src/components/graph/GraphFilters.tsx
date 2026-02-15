import { Badge } from "../ui/badge";
import { Button } from "../ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../ui/select";

interface Props {
  kinds: string[];
  selectedKinds: string[];
  onKindsChange: (kinds: string[]) => void;
  depth: number;
  onDepthChange: (depth: number) => void;
  direction: string;
  onDirectionChange: (direction: string) => void;
}

const ALL_KINDS = [
  "table",
  "view",
  "procedure",
  "function",
  "trigger",
  "class",
  "interface",
  "method",
  "column",
];

export function GraphFilters({
  selectedKinds,
  onKindsChange,
  depth,
  onDepthChange,
  direction,
  onDirectionChange,
}: Props) {
  const toggleKind = (kind: string) => {
    if (selectedKinds.includes(kind)) {
      onKindsChange(selectedKinds.filter((k) => k !== kind));
    } else {
      onKindsChange([...selectedKinds, kind]);
    }
  };

  return (
    <div className="space-y-3 border-b bg-card px-4 py-3">
      <div className="flex items-center gap-2">
        <span className="text-xs font-medium text-muted-foreground">Direction:</span>
        {["upstream", "downstream", "both"].map((d) => (
          <Button
            key={d}
            variant={direction === d ? "default" : "ghost"}
            size="sm"
            onClick={() => onDirectionChange(d)}
          >
            {d}
          </Button>
        ))}
        <span className="ml-4 text-xs font-medium text-muted-foreground">Depth:</span>
        <Select value={depth.toString()} onValueChange={(v) => onDepthChange(Number(v))}>
          <SelectTrigger className="w-16">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {[1, 2, 3, 4, 5].map((d) => (
              <SelectItem key={d} value={d.toString()}>
                {d}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="flex flex-wrap gap-1">
        {ALL_KINDS.map((kind) => (
          <Badge
            key={kind}
            variant={
              selectedKinds.length === 0 || selectedKinds.includes(kind) ? "default" : "outline"
            }
            className="cursor-pointer capitalize"
            onClick={() => toggleKind(kind)}
          >
            {kind}
          </Badge>
        ))}
      </div>
    </div>
  );
}
