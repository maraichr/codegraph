import { useState, useCallback } from "react";
import { useParams } from "react-router";
import { useSymbolSearch, useSymbolLineage } from "../api/hooks";
import type { LineageGraph } from "../api/types";
import { CytoscapeGraph } from "../components/graph/CytoscapeGraph";
import { GraphToolbar } from "../components/graph/GraphToolbar";
import { GraphFilters } from "../components/graph/GraphFilters";
import { NodeDetail } from "../components/graph/NodeDetail";

export function GraphExplorer() {
  const { slug } = useParams<{ slug: string }>();
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedSymbolId, setSelectedSymbolId] = useState<string | null>(null);
  const [clickedNodeId, setClickedNodeId] = useState<string | null>(null);
  const [layout, setLayout] = useState("dagre");
  const [direction, setDirection] = useState("both");
  const [depth, setDepth] = useState(3);
  const [selectedKinds, setSelectedKinds] = useState<string[]>([]);

  const { data: searchResults } = useSymbolSearch(
    slug ?? "",
    searchQuery,
    selectedKinds.length > 0 ? selectedKinds : undefined,
  );

  const { data: lineageData } = useSymbolLineage(
    selectedSymbolId ?? "",
    direction,
    depth,
  );

  const graph: LineageGraph | null = lineageData ?? null;

  const handleNodeClick = useCallback((id: string) => {
    setClickedNodeId(id);
  }, []);

  const handleSymbolSelect = (symbolId: string) => {
    setSelectedSymbolId(symbolId);
    setClickedNodeId(null);
  };

  if (!slug) return null;

  return (
    <div className="flex h-[calc(100vh-4rem)] flex-col">
      <div className="flex items-center gap-4 border-b border-gray-200 bg-white px-4 py-3">
        <h2 className="text-lg font-semibold text-gray-900">Graph Explorer</h2>
        <div className="flex-1">
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search symbols..."
            className="w-full max-w-md rounded-md border border-gray-300 px-3 py-1.5 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
      </div>

      <GraphFilters
        kinds={[]}
        selectedKinds={selectedKinds}
        onKindsChange={setSelectedKinds}
        depth={depth}
        onDepthChange={setDepth}
        direction={direction}
        onDirectionChange={setDirection}
      />

      <GraphToolbar
        layout={layout}
        onLayoutChange={setLayout}
        onFit={() => {}}
      />

      <div className="flex flex-1 overflow-hidden">
        {/* Search results sidebar */}
        {searchQuery.length >= 2 && searchResults && (
          <div className="w-64 overflow-y-auto border-r border-gray-200 bg-gray-50 p-2">
            <p className="mb-2 text-xs font-medium text-gray-500">
              {searchResults.count} result{searchResults.count !== 1 ? "s" : ""}
            </p>
            <ul className="space-y-1">
              {searchResults.symbols.map((sym) => (
                <li key={sym.id}>
                  <button
                    type="button"
                    onClick={() => handleSymbolSelect(sym.id)}
                    className={`w-full rounded px-2 py-1.5 text-left text-xs hover:bg-white ${
                      selectedSymbolId === sym.id
                        ? "bg-white ring-1 ring-blue-300"
                        : ""
                    }`}
                  >
                    <div className="font-medium text-gray-900">{sym.name}</div>
                    <div className="text-gray-500">
                      <span className="inline-block rounded bg-gray-100 px-1 text-[10px]">
                        {sym.kind}
                      </span>{" "}
                      {sym.qualified_name}
                    </div>
                  </button>
                </li>
              ))}
            </ul>
          </div>
        )}

        {/* Graph canvas */}
        <div className="flex-1">
          {graph ? (
            <CytoscapeGraph
              graph={graph}
              layout={layout}
              onNodeClick={handleNodeClick}
            />
          ) : (
            <div className="flex h-full items-center justify-center text-gray-400">
              <p className="text-sm">
                {selectedSymbolId
                  ? "Loading graph..."
                  : "Search for a symbol and select it to view its dependency graph."}
              </p>
            </div>
          )}
        </div>

        {/* Node detail panel */}
        <NodeDetail
          nodeId={clickedNodeId}
          graph={graph}
          onClose={() => setClickedNodeId(null)}
        />
      </div>
    </div>
  );
}
