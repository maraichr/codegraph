import { useState, useCallback, useEffect, useRef } from "react";
import { useParams } from "react-router";
import cytoscape from "cytoscape";
import { useSymbolSearch, useColumnLineage } from "../api/hooks";
import type { ColumnLineageGraph, ColumnLineageNode } from "../api/types";
import { LineageControls } from "../components/lineage/LineageControls";
import { ColumnNodeTooltip } from "../components/lineage/ColumnNodeTooltip";

const DERIVATION_STYLES: Record<string, { color: string; style: string }> = {
  direct_copy: { color: "#3b82f6", style: "solid" },
  transform: { color: "#f59e0b", style: "dashed" },
  transforms_to: { color: "#f59e0b", style: "dashed" },
  aggregate: { color: "#8b5cf6", style: "dotted" },
  filter: { color: "#10b981", style: "solid" },
  join: { color: "#ef4444", style: "solid" },
  uses_column: { color: "#6b7280", style: "solid" },
};

const ALL_DERIVATIONS = [
  "direct_copy",
  "transform",
  "aggregate",
  "filter",
  "join",
];

export function LineageExplorer() {
  const { slug } = useParams<{ slug: string }>();
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedColumnId, setSelectedColumnId] = useState<string | null>(null);
  const [clickedNode, setClickedNode] = useState<ColumnLineageNode | null>(
    null,
  );
  const [direction, setDirection] = useState("both");
  const [depth, setDepth] = useState(5);
  const [selectedDerivations, setSelectedDerivations] = useState<string[]>(
    ALL_DERIVATIONS,
  );

  const containerRef = useRef<HTMLDivElement>(null);
  const cyRef = useRef<cytoscape.Core | null>(null);

  const { data: searchResults } = useSymbolSearch(slug ?? "", searchQuery, [
    "column",
  ]);
  const { data: lineageData } = useColumnLineage(
    selectedColumnId ?? "",
    direction,
    depth,
  );

  // Initialize cytoscape
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
            "font-size": "10px",
            "text-wrap": "wrap",
            "text-max-width": "80px",
            "text-valign": "bottom",
            "text-margin-y": 4,
            width: 30,
            height: 30,
            shape: "ellipse",
          },
        },
        {
          selector: "node.table-group",
          style: {
            shape: "round-rectangle",
            "background-color": "#f3f4f6",
            "border-width": 1,
            "border-color": "#d1d5db",
            "font-weight": "bold",
            "font-size": "11px",
          },
        },
        {
          selector: "node:selected",
          style: { "border-width": 3, "border-color": "#2563eb" },
        },
        {
          selector: "node.highlighted",
          style: { "border-width": 4, "border-color": "#f59e0b" },
        },
        {
          selector: "edge",
          style: {
            width: 2,
            "target-arrow-shape": "triangle",
            "curve-style": "bezier",
            label: "data(derivationType)",
            "font-size": "8px",
            color: "#9ca3af",
          },
        },
        {
          selector: "edge.direct_copy",
          style: { "line-color": "#3b82f6", "target-arrow-color": "#3b82f6" },
        },
        {
          selector: "edge.transform, edge.transforms_to",
          style: {
            "line-color": "#f59e0b",
            "target-arrow-color": "#f59e0b",
            "line-style": "dashed",
          },
        },
        {
          selector: "edge.aggregate",
          style: {
            "line-color": "#8b5cf6",
            "target-arrow-color": "#8b5cf6",
            "line-style": "dotted",
          },
        },
      ],
      layout: { name: "grid" },
      minZoom: 0.2,
      maxZoom: 5,
    });

    cyRef.current = cy;

    cy.on("tap", "node", (evt) => {
      const data = evt.target.data();
      if (data.nodeData) {
        setClickedNode(data.nodeData);
      }
    });

    return () => {
      cy.destroy();
    };
  }, []);

  // Update graph when lineage data changes
  useEffect(() => {
    const cy = cyRef.current;
    if (!cy || !lineageData) return;

    cy.elements().remove();

    const graph: ColumnLineageGraph = lineageData;
    const elements: cytoscape.ElementDefinition[] = [];

    for (const node of graph.nodes) {
      elements.push({
        data: {
          id: node.id,
          label: `${node.table_name ? node.table_name + "." : ""}${node.name}`,
          color: node.kind === "column" ? "#3b82f6" : "#6b7280",
          nodeData: node,
        },
      });
    }

    for (const edge of graph.edges) {
      if (
        selectedDerivations.length > 0 &&
        !selectedDerivations.includes(edge.derivation_type)
      ) {
        continue;
      }

      const style = DERIVATION_STYLES[edge.derivation_type] ?? {
        color: "#6b7280",
        style: "solid",
      };
      elements.push({
        data: {
          source: edge.source_id,
          target: edge.target_id,
          derivationType: edge.derivation_type.replace(/_/g, " "),
          color: style.color,
        },
        classes: edge.derivation_type,
      });
    }

    if (elements.length === 0) return;

    cy.add(elements);

    // Highlight root
    const root = cy.getElementById(graph.root_column_id);
    if (root.length > 0) {
      root.addClass("highlighted");
    }

    // Run layout only when there are nodes
    if (cy.nodes().length > 0) {
      cy.layout({
        name: "breadthfirst",
        directed: true,
        animate: true,
        animationDuration: 500,
      } as cytoscape.LayoutOptions).run();

      cy.fit(undefined, 40);
    }
  }, [lineageData, selectedDerivations]);

  const handleColumnSelect = useCallback((columnId: string) => {
    setSelectedColumnId(columnId);
    setClickedNode(null);
  }, []);

  if (!slug) return null;

  return (
    <div className="flex h-[calc(100vh-4rem)] flex-col">
      <div className="flex items-center gap-4 border-b border-gray-200 bg-white px-4 py-3">
        <h2 className="text-lg font-semibold text-gray-900">
          Column Lineage Explorer
        </h2>
      </div>

      <LineageControls
        direction={direction}
        onDirectionChange={setDirection}
        depth={depth}
        onDepthChange={setDepth}
        derivationTypes={ALL_DERIVATIONS}
        selectedDerivations={selectedDerivations}
        onDerivationsChange={setSelectedDerivations}
      />

      <div className="flex flex-1 overflow-hidden">
        {/* Column search sidebar — always visible */}
        <div className="flex w-72 flex-col border-r border-gray-200 bg-gray-50">
          <div className="border-b border-gray-200 p-3">
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search columns..."
              className="w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>
          <div className="flex-1 overflow-y-auto p-2">
            {searchQuery.length < 2 ? (
              <p className="px-2 py-4 text-center text-xs text-gray-400">
                Type at least 2 characters to search columns.
              </p>
            ) : !searchResults ? (
              <p className="px-2 py-4 text-center text-xs text-gray-400">
                Searching...
              </p>
            ) : searchResults.count === 0 ? (
              <p className="px-2 py-4 text-center text-xs text-gray-400">
                No columns found.
              </p>
            ) : (
              <>
                <p className="mb-2 text-xs font-medium text-gray-500">
                  {searchResults.count} column
                  {searchResults.count !== 1 ? "s" : ""}
                </p>
                <ul className="space-y-1">
                  {searchResults.symbols.map((sym) => (
                    <li key={sym.id}>
                      <button
                        type="button"
                        onClick={() => handleColumnSelect(sym.id)}
                        className={`w-full rounded px-2 py-1.5 text-left text-xs hover:bg-white ${
                          selectedColumnId === sym.id
                            ? "bg-white ring-1 ring-blue-300"
                            : ""
                        }`}
                      >
                        <div className="font-medium text-gray-900">
                          {sym.name}
                        </div>
                        <div className="truncate text-gray-500">
                          {sym.qualified_name}
                        </div>
                      </button>
                    </li>
                  ))}
                </ul>
              </>
            )}
          </div>
        </div>

        {/* Graph canvas */}
        <div className="relative flex-1">
          <div
            ref={containerRef}
            className="h-full w-full"
            style={{ minHeight: "500px" }}
          />
          {!lineageData ? (
            <div className="absolute inset-0 flex items-center justify-center bg-white text-gray-400">
              <p className="text-sm">
                {selectedColumnId
                  ? "Loading column lineage..."
                  : "Select a column from the sidebar to view its data flow lineage."}
              </p>
            </div>
          ) : lineageData.nodes.length === 0 ? (
            <div className="absolute inset-0 flex items-center justify-center bg-white text-gray-400">
              <p className="text-sm">
                No column lineage found for this column.
              </p>
            </div>
          ) : null}
        </div>

        {/* Detail panel */}
        <ColumnNodeTooltip
          node={clickedNode}
          edges={lineageData?.edges ?? []}
          onClose={() => setClickedNode(null)}
        />
      </div>
    </div>
  );
}
