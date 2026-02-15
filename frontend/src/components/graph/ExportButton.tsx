import { Download } from "lucide-react";
import { Button } from "../ui/button";
import type { CytoscapeGraphHandle } from "./CytoscapeGraph";

interface Props {
  graphRef: React.RefObject<CytoscapeGraphHandle | null>;
}

export function ExportButton({ graphRef }: Props) {
  const handleExport = () => {
    const cy = graphRef.current?.cy;
    if (!cy) return;

    const png = cy.png({ full: true, scale: 2, bg: "#ffffff" });
    const link = document.createElement("a");
    link.href = png;
    link.download = "codegraph-export.png";
    link.click();
  };

  return (
    <Button variant="outline" size="sm" onClick={handleExport}>
      <Download className="h-4 w-4" />
      Export PNG
    </Button>
  );
}
