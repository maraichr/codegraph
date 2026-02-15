import { Maximize2 } from "lucide-react";
import { Button } from "../ui/button";

interface Props {
  layout: string;
  onLayoutChange: (layout: string) => void;
  onFit: () => void;
  children?: React.ReactNode;
}

export function GraphToolbar({ layout, onLayoutChange, onFit, children }: Props) {
  return (
    <div className="flex items-center gap-2 border-b bg-card px-4 py-2">
      <span className="text-xs font-medium text-muted-foreground">Layout:</span>
      {["dagre", "cose", "breadthfirst"].map((l) => (
        <Button
          key={l}
          variant={layout === l ? "default" : "ghost"}
          size="sm"
          onClick={() => onLayoutChange(l)}
        >
          {l}
        </Button>
      ))}
      <div className="ml-auto flex items-center gap-2">
        {children}
        <Button variant="outline" size="sm" onClick={onFit}>
          <Maximize2 className="h-4 w-4" />
          Fit
        </Button>
      </div>
    </div>
  );
}
