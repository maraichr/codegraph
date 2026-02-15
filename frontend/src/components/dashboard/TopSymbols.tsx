import { useState } from "react";
import type { TopSymbol } from "../../api/types";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import { Skeleton } from "../ui/skeleton";

interface Props {
  byInDegree: TopSymbol[] | undefined;
  byPageRank: TopSymbol[] | undefined;
  isLoading: boolean;
}

export function TopSymbols({ byInDegree, byPageRank, isLoading }: Props) {
  const [mode, setMode] = useState<"in-degree" | "pagerank">("in-degree");

  const symbols = mode === "in-degree" ? byInDegree : byPageRank;

  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between space-y-0">
        <CardTitle className="text-sm">Top Symbols</CardTitle>
        <div className="flex gap-1">
          <Button
            variant={mode === "in-degree" ? "default" : "ghost"}
            size="sm"
            onClick={() => setMode("in-degree")}
          >
            In-Degree
          </Button>
          <Button
            variant={mode === "pagerank" ? "default" : "ghost"}
            size="sm"
            onClick={() => setMode("pagerank")}
          >
            PageRank
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={`skel-${i.toString()}`} className="h-8 w-full" />
            ))}
          </div>
        ) : !symbols?.length ? (
          <p className="text-sm text-muted-foreground">No analytics data available yet.</p>
        ) : (
          <div className="space-y-2">
            {symbols.map((sym) => (
              <div key={sym.id} className="flex items-center justify-between rounded-md border p-2">
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium">{sym.name}</p>
                  <p className="truncate text-xs text-muted-foreground">{sym.qualified_name}</p>
                </div>
                <div className="flex items-center gap-2">
                  <Badge variant="outline" className="text-xs">
                    {sym.kind}
                  </Badge>
                  <span className="text-sm font-mono text-muted-foreground">
                    {mode === "in-degree" ? sym.in_degree : sym.pagerank?.toFixed(4)}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
