import { useEffect, useRef } from "react";
import cytoscape from "cytoscape";
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

interface Props {
  graph: LineageGraph | null;
  layout?: string;
  onNodeClick?: (id: string) => void;
}

export function CytoscapeGraph({ graph, layout = "dagre", onNodeClick }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const cyRef = useRef<cytoscape.Core | null>(null);

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
      elements.push({
        data: {
          id: node.ID,
          label: node.Name,
          kind: node.Kind,
          qualifiedName: node.QualifiedName,
          language: node.Language,
          color: KIND_COLORS[node.Kind.toLowerCase()] ?? "#6b7280",
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
    <div
      ref={containerRef}
      className="h-full w-full"
      style={{ minHeight: "500px" }}
    />
  );
}
