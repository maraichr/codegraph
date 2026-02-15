import type { ColumnLineageGraph, ColumnLineageNode } from "../../api/types";
import { Badge } from "../ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";

interface Props {
  graph: ColumnLineageGraph | null;
  selectedNodeId: string | null;
  onColumnClick: (columnId: string) => void;
}

export function ColumnLineagePanel({ graph, selectedNodeId, onColumnClick }: Props) {
  if (!graph || !selectedNodeId) return null;

  const selectedNode = graph.nodes.find((n: ColumnLineageNode) => n.id === selectedNodeId);
  if (!selectedNode || selectedNode.kind !== "table") return null;

  const columns = graph.nodes.filter(
    (n: ColumnLineageNode) => n.kind === "column" && n.table_name === selectedNode.name,
  );

  if (columns.length === 0) return null;

  return (
    <Card className="w-64">
      <CardHeader className="pb-2">
        <CardTitle className="text-sm">Columns</CardTitle>
        <p className="text-xs text-muted-foreground">{selectedNode.name}</p>
      </CardHeader>
      <CardContent className="space-y-1">
        {columns.map((col: ColumnLineageNode) => {
          const hasEdges = graph.edges.some(
            (e) => e.source_id === col.id || e.target_id === col.id,
          );
          return (
            <button
              key={col.id}
              type="button"
              onClick={() => onColumnClick(col.id)}
              className="flex w-full items-center justify-between rounded-md px-2 py-1 text-left text-sm hover:bg-accent"
            >
              <span className="truncate">{col.name}</span>
              {hasEdges && (
                <Badge variant="secondary" className="text-[10px]">
                  lineage
                </Badge>
              )}
            </button>
          );
        })}
      </CardContent>
    </Card>
  );
}
