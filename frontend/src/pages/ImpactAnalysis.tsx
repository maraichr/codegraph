import { useState } from "react";
import { useParams } from "react-router";
import { useImpactAnalysis, useSymbolSearch } from "../api/hooks";
import { ImpactSummary } from "../components/impact/ImpactSummary";
import { ImpactTree } from "../components/impact/ImpactTree";

const CHANGE_TYPES = [
  { value: "modify", label: "Modify" },
  { value: "delete", label: "Delete" },
  { value: "rename", label: "Rename" },
];

export function ImpactAnalysis() {
  const { slug } = useParams<{ slug: string }>();
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedSymbolId, setSelectedSymbolId] = useState<string | null>(null);
  const [changeType, setChangeType] = useState("modify");
  const [maxDepth, setMaxDepth] = useState(5);
  const [viewMode, setViewMode] = useState<"tree" | "list">("tree");

  const { data: searchResults } = useSymbolSearch(slug ?? "", searchQuery);
  const { data: impactData } = useImpactAnalysis(selectedSymbolId ?? "", changeType, maxDepth);

  if (!slug) return null;

  return (
    <div className="flex h-[calc(100vh-4rem)] flex-col">
      {/* Header */}
      <div className="flex items-center gap-4 border-b border-gray-200 bg-white px-4 py-3">
        <h2 className="text-lg font-semibold text-gray-900">Impact Analysis</h2>
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

      {/* Controls */}
      <div className="flex items-center gap-4 border-b border-gray-200 bg-gray-50 px-4 py-2">
        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-gray-600">Change Type:</span>
          <div className="flex rounded-md border border-gray-300">
            {CHANGE_TYPES.map((ct) => (
              <button
                key={ct.value}
                type="button"
                onClick={() => setChangeType(ct.value)}
                className={`px-2.5 py-1 text-xs font-medium ${
                  changeType === ct.value
                    ? ct.value === "delete"
                      ? "bg-red-600 text-white"
                      : "bg-blue-600 text-white"
                    : "bg-white text-gray-700 hover:bg-gray-50"
                } first:rounded-l-md last:rounded-r-md`}
              >
                {ct.label}
              </button>
            ))}
          </div>
        </div>

        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-gray-600">Max Depth:</span>
          <input
            type="range"
            min={1}
            max={10}
            value={maxDepth}
            onChange={(e) => setMaxDepth(parseInt(e.target.value, 10))}
            className="h-1.5 w-24"
          />
          <span className="w-4 text-xs text-gray-500">{maxDepth}</span>
        </div>

        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-gray-600">View:</span>
          <div className="flex rounded-md border border-gray-300">
            <button
              type="button"
              onClick={() => setViewMode("tree")}
              className={`px-2.5 py-1 text-xs font-medium ${
                viewMode === "tree"
                  ? "bg-blue-600 text-white"
                  : "bg-white text-gray-700 hover:bg-gray-50"
              } rounded-l-md`}
            >
              Tree
            </button>
            <button
              type="button"
              onClick={() => setViewMode("list")}
              className={`px-2.5 py-1 text-xs font-medium ${
                viewMode === "list"
                  ? "bg-blue-600 text-white"
                  : "bg-white text-gray-700 hover:bg-gray-50"
              } rounded-r-md`}
            >
              List
            </button>
          </div>
        </div>
      </div>

      {/* Summary bar */}
      {impactData && (
        <ImpactSummary
          totalAffected={impactData.total_affected}
          directImpact={impactData.direct_impact}
          transitiveImpact={impactData.transitive_impact}
        />
      )}

      <div className="flex flex-1 overflow-hidden">
        {/* Search results sidebar */}
        {searchQuery.length >= 2 && searchResults && (
          <div className="w-64 overflow-y-auto border-r border-gray-200 bg-gray-50 p-2">
            <p className="mb-2 text-xs font-medium text-gray-500">
              {searchResults.count} result
              {searchResults.count !== 1 ? "s" : ""}
            </p>
            <ul className="space-y-1">
              {searchResults.symbols.map((sym) => (
                <li key={sym.id}>
                  <button
                    type="button"
                    onClick={() => {
                      setSelectedSymbolId(sym.id);
                    }}
                    className={`w-full rounded px-2 py-1.5 text-left text-xs hover:bg-white ${
                      selectedSymbolId === sym.id ? "bg-white ring-1 ring-blue-300" : ""
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

        {/* Impact results */}
        <div className="flex-1 overflow-y-auto">
          {impactData ? (
            <ImpactTree
              directImpact={impactData.direct_impact}
              transitiveImpact={impactData.transitive_impact}
              viewMode={viewMode}
            />
          ) : (
            <div className="flex h-full items-center justify-center text-gray-400">
              <p className="text-sm">
                {selectedSymbolId
                  ? "Loading impact analysis..."
                  : "Search for a symbol and select it to analyze the impact of changes."}
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
