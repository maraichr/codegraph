import { X } from "lucide-react";
import type { LineageGraph, LineageNode } from "../../api/types";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";

interface Props {
  nodeId: string | null;
  graph: LineageGraph | null;
  onClose: () => void;
}

export function NodeDetail({ nodeId, graph, onClose }: Props) {
  if (!nodeId || !graph) return null;

  const node = graph.Nodes.find((n: LineageNode) => n.ID === nodeId);
  if (!node) return null;

  const incoming = graph.Edges.filter((e) => e.TargetID === nodeId);
  const outgoing = graph.Edges.filter((e) => e.SourceID === nodeId);

  const getNodeName = (id: string) => {
    const n = graph.Nodes.find((x: LineageNode) => x.ID === id);
    return n?.Name ?? id.slice(0, 8);
  };

  return (
    <div className="w-72 overflow-y-auto border-l bg-card p-4">
      <div className="mb-3 flex items-center justify-between">
        <h4 className="text-sm font-semibold">Symbol Detail</h4>
        <Button variant="ghost" size="icon" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      <dl className="space-y-2 text-xs">
        <div>
          <dt className="font-medium text-muted-foreground">Name</dt>
          <dd className="font-mono">{node.Name}</dd>
        </div>
        <div>
          <dt className="font-medium text-muted-foreground">Qualified Name</dt>
          <dd className="font-mono">{node.QualifiedName}</dd>
        </div>
        <div>
          <dt className="font-medium text-muted-foreground">Kind</dt>
          <dd>
            <Badge variant="outline">{node.Kind}</Badge>
          </dd>
        </div>
        <div>
          <dt className="font-medium text-muted-foreground">Language</dt>
          <dd>{node.Language}</dd>
        </div>
      </dl>

      {incoming.length > 0 && (
        <div className="mt-4">
          <h5 className="text-xs font-medium text-muted-foreground">
            Incoming ({incoming.length})
          </h5>
          <ul className="mt-1 space-y-1">
            {incoming.map((e) => (
              <li key={`${e.SourceID}-${e.TargetID}-${e.EdgeType}`} className="text-xs">
                <span className="font-mono">{getNodeName(e.SourceID)}</span>
                <span className="mx-1 text-muted-foreground">{e.EdgeType}</span>
              </li>
            ))}
          </ul>
        </div>
      )}

      {outgoing.length > 0 && (
        <div className="mt-4">
          <h5 className="text-xs font-medium text-muted-foreground">
            Outgoing ({outgoing.length})
          </h5>
          <ul className="mt-1 space-y-1">
            {outgoing.map((e) => (
              <li key={`${e.SourceID}-${e.TargetID}-${e.EdgeType}`} className="text-xs">
                <span className="mx-1 text-muted-foreground">{e.EdgeType}</span>
                <span className="font-mono">{getNodeName(e.TargetID)}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
