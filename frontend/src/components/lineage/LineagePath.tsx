import { ChevronRight } from "lucide-react";

interface Props {
  path: string[];
}

export function LineagePath({ path }: Props) {
  if (path.length === 0) return null;

  return (
    <div className="flex items-center gap-1 text-sm text-muted-foreground">
      {path.map((step, i) => (
        <span key={`${step}-${i.toString()}`} className="flex items-center gap-1">
          {i > 0 && <ChevronRight className="h-3 w-3" />}
          <span className="font-mono text-xs">{step}</span>
        </span>
      ))}
    </div>
  );
}
