import { useState } from "react";
import { useParams } from "react-router";
import { useImpactAnalysis, useSymbolSearch } from "../api/hooks";
import { ImpactSummary } from "../components/impact/ImpactSummary";
import { ImpactTree } from "../components/impact/ImpactTree";
import { Input } from "../components/ui/input";

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
      <div className="flex items-center gap-4 border-b border-border bg-card px-4 py-3">
        <h2 className="text-lg font-semibold text-foreground">Impact Analysis</h2>
        <div className="flex-1">
          <Input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search symbols..."
            className="max-w-md"
          />
        </div>
      </div>

      {/* Controls */}
      <div className="flex items-center gap-4 border-b border-border bg-muted px-4 py-2">
        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-muted-foreground">Change Type:</span>
          <div className="flex rounded-md border border-border">
            {CHANGE_TYPES.map((ct) => (
              <button
                key={ct.value}
                type="button"
                onClick={() => setChangeType(ct.value)}
                className={`px-2.5 py-1 text-xs font-medium transition-colors ${
                  changeType === ct.value
                    ? ct.value === "delete"
                      ? "bg-destructive text-destructive-foreground"
                      : "bg-primary text-primary-foreground"
                    : "bg-secondary text-secondary-foreground hover:bg-accent"
                } first:rounded-l-md last:rounded-r-md`}
              >
                {ct.label}
              </button>
            ))}
          </div>
        </div>

        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-muted-foreground">Max Depth:</span>
          <input
            type="range"
            min={1}
            max={10}
            value={maxDepth}
            onChange={(e) => setMaxDepth(parseInt(e.target.value, 10))}
            className="h-1.5 w-24 accent-primary"
          />
          <span className="w-4 text-xs text-muted-foreground">{maxDepth}</span>
        </div>

        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-muted-foreground">View:</span>
          <div className="flex rounded-md border border-border">
            <button
              type="button"
              onClick={() => setViewMode("tree")}
              className={`px-2.5 py-1 text-xs font-medium transition-colors ${
                viewMode === "tree"
                  ? "bg-primary text-primary-foreground"
                  : "bg-secondary text-secondary-foreground hover:bg-accent"
              } rounded-l-md`}
            >
              Tree
            </button>
            <button
              type="button"
              onClick={() => setViewMode("list")}
              className={`px-2.5 py-1 text-xs font-medium transition-colors ${
                viewMode === "list"
                  ? "bg-primary text-primary-foreground"
                  : "bg-secondary text-secondary-foreground hover:bg-accent"
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
          <div className="w-64 overflow-y-auto border-r border-border bg-muted p-2">
            <p className="mb-2 text-xs font-medium text-muted-foreground">
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
                    className={`w-full rounded px-2 py-1.5 text-left text-xs transition-colors hover:bg-accent ${
                      selectedSymbolId === sym.id ? "bg-accent ring-1 ring-primary" : ""
                    }`}
                  >
                    <div className="font-medium text-foreground">{sym.name}</div>
                    <div className="text-muted-foreground">
                      <span className="inline-block rounded bg-secondary px-1 text-[10px]">
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
            <div className="flex h-full items-center justify-center text-muted-foreground">
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
