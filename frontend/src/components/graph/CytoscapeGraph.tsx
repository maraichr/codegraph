import cytoscape from "cytoscape";
import { forwardRef, useEffect, useImperativeHandle, useRef, useState } from "react";
import type { LineageGraph } from "../../api/types";

const KIND_COLORS: Record<string, string> = {
  table: "#3b82f6",
  view: "#8b5cf6",
  procedure: "#10b981",
  stored_procedure: "#10b981",
  function: "#f59e0b",
  trigger: "#ef4444",
  column: "#6b7280",
  class: "#06b6d4",
  interface: "#a855f7",
  method: "#84cc16",
  package: "#f97316",
};

export interface CytoscapeGraphHandle {
  cy: cytoscape.Core | null;
  fit: () => void;
}

interface Props {
  graph: LineageGraph | null;
  layout?: string;
  onNodeClick?: (id: string) => void;
}

export const CytoscapeGraph = forwardRef<CytoscapeGraphHandle, Props>(function CytoscapeGraph(
  { graph, layout = "dagre", onNodeClick },
  ref,
) {
  const containerRef = useRef<HTMLDivElement>(null);
  const cyRef = useRef<cytoscape.Core | null>(null);
  const [tooltip, setTooltip] = useState<{ x: number; y: number; text: string } | null>(null);

  useImperativeHandle(ref, () => ({
    cy: cyRef.current,
    fit: () => cyRef.current?.fit(undefined, 40),
  }));

  useEffect(() => {
    if (!containerRef.current) return;

    const cy = cytoscape({
      container: containerRef.current,
      style: [
        {
          selector: "node",
          style: {
            label: "data(label)",
            "background-color": "data(color)",
            color: "#1f2937",
            "font-size": "11px",
            "text-wrap": "wrap",
            "text-max-width": "100px",
            "text-valign": "bottom",
            "text-margin-y": 4,
            width: 36,
            height: 36,
          },
        },
        {
          selector: "node:selected",
          style: {
            "border-width": 3,
            "border-color": "#2563eb",
          },
        },
        {
          selector: "edge",
          style: {
            label: "data(edgeType)",
            "font-size": "9px",
            color: "#9ca3af",
            width: 2,
            "line-color": "#d1d5db",
            "target-arrow-color": "#9ca3af",
            "target-arrow-shape": "triangle",
            "curve-style": "bezier",
          },
        },
        {
          selector: "node.highlighted",
          style: {
            "border-width": 4,
            "border-color": "#f59e0b",
          },
        },
      ],
      layout: { name: "grid" },
      minZoom: 0.2,
      maxZoom: 5,
    });

    cyRef.current = cy;

    cy.on("tap", "node", (evt) => {
      const id = evt.target.data("id");
      if (onNodeClick) onNodeClick(id);
    });

    cy.on("mouseover", "node", (evt) => {
      const node = evt.target;
      const pos = node.renderedPosition();
      const data = node.data();
      const parts = [data.label, `Kind: ${data.kind}`, `Language: ${data.language ?? "unknown"}`];
      if (data.inDegree != null) parts.push(`In-degree: ${data.inDegree}`);
      if (data.pagerank != null) parts.push(`PageRank: ${Number(data.pagerank).toFixed(4)}`);
      if (data.layer) parts.push(`Layer: ${data.layer}`);
      setTooltip({ x: pos.x, y: pos.y - 30, text: parts.join("\n") });
    });

    cy.on("mouseout", "node", () => {
      setTooltip(null);
    });

    return () => {
      cy.destroy();
    };
  }, [onNodeClick]);

  useEffect(() => {
    const cy = cyRef.current;
    if (!cy || !graph) return;

    cy.elements().remove();

    const elements: cytoscape.ElementDefinition[] = [];

    for (const node of graph.Nodes) {
      const meta = node.Metadata as Record<string, unknown> | undefined;
      elements.push({
        data: {
          id: node.ID,
          label: node.Name,
          kind: node.Kind,
          qualifiedName: node.QualifiedName,
          language: node.Language,
          color: KIND_COLORS[node.Kind.toLowerCase()] ?? "#6b7280",
          inDegree: meta?.in_degree ?? null,
          pagerank: meta?.pagerank ?? null,
          layer: meta?.layer ?? null,
        },
      });
    }

    for (const edge of graph.Edges) {
      elements.push({
        data: {
          source: edge.SourceID,
          target: edge.TargetID,
          edgeType: edge.EdgeType,
        },
      });
    }

    cy.add(elements);

    // Highlight root node
    const root = cy.getElementById(graph.RootID);
    if (root.length > 0) {
      root.addClass("highlighted");
    }

    // Apply layout
    cy.layout({
      name: layout === "dagre" ? "breadthfirst" : layout === "cose" ? "cose" : "breadthfirst",
      animate: true,
      animationDuration: 500,
    } as cytoscape.LayoutOptions).run();

    cy.fit(undefined, 40);
  }, [graph, layout]);

  return (
    <div className="relative h-full w-full" style={{ minHeight: "500px" }}>
      <div ref={containerRef} className="h-full w-full" />
      {tooltip && (
        <div
          className="pointer-events-none absolute z-50 max-w-xs whitespace-pre rounded-md border bg-popover px-3 py-2 text-xs text-popover-foreground shadow-md"
          style={{ left: tooltip.x, top: tooltip.y, transform: "translate(-50%, -100%)" }}
        >
          {tooltip.text}
        </div>
      )}
    </div>
  );
});
