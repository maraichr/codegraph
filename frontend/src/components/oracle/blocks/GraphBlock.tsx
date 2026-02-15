import { useRef } from "react";
import { CytoscapeGraph } from "../../graph/CytoscapeGraph";
import type { OracleGraphData } from "../../../api/types";
import type { LineageGraph } from "../../../api/types";

interface Props {
  data: unknown;
}

export function GraphBlock({ data }: Props) {
  const d = data as OracleGraphData;
  const graphRef = useRef(null);

  if (!d.nodes?.length) return null;

  // Convert Oracle graph data to LineageGraph format for CytoscapeGraph
  const lineageGraph: LineageGraph = {
    Nodes: d.nodes.map((n) => ({
      ID: n.id,
      Name: n.name,
      QualifiedName: n.name,
      Kind: n.kind,
      Language: "",
      FileID: "",
    })),
    Edges: d.edges?.map((e) => ({
      SourceID: e.source,
      TargetID: e.target,
      EdgeType: e.edge_type,
    })) || [],
    RootID: d.nodes[0]?.id || "",
  };

  return (
    <div className="rounded-md border border-border/50 bg-muted/10 overflow-hidden">
      <div className="h-[200px]">
        <CytoscapeGraph ref={graphRef} graph={lineageGraph} layout="cose" />
      </div>
      <div className="border-t border-border/50 px-2.5 py-1.5 text-[10px] text-muted-foreground">
        {d.nodes.length} nodes, {d.edges?.length || 0} edges
      </div>
    </div>
  );
}
