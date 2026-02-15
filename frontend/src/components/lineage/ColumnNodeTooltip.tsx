import type { ColumnLineageEdge, ColumnLineageNode } from "../../api/types";

interface Props {
  node: ColumnLineageNode | null;
  edges: ColumnLineageEdge[];
  onClose: () => void;
}

export function ColumnNodeTooltip({ node, edges, onClose }: Props) {
  if (!node) return null;

  const incomingEdges = edges.filter((e) => e.target_id === node.id);
  const outgoingEdges = edges.filter((e) => e.source_id === node.id);

  return (
    <div className="w-72 overflow-y-auto border-l border-border bg-card p-4">
      <div className="mb-3 flex items-center justify-between">
        <h4 className="text-sm font-semibold text-foreground">Column Details</h4>
        <button
          type="button"
          onClick={onClose}
          className="text-muted-foreground hover:text-foreground"
        >
          &times;
        </button>
      </div>

      <div className="space-y-3">
        <div>
          <p className="text-xs font-medium text-muted-foreground">Name</p>
          <p className="text-sm font-medium text-foreground">{node.name}</p>
        </div>

        <div>
          <p className="text-xs font-medium text-muted-foreground">Table</p>
          <p className="text-sm text-secondary-foreground">{node.table_name || "N/A"}</p>
        </div>

        <div>
          <p className="text-xs font-medium text-muted-foreground">Qualified Name</p>
          <p className="break-all font-mono text-xs text-muted-foreground">{node.qualified_name}</p>
        </div>

        <div>
          <p className="text-xs font-medium text-muted-foreground">Kind</p>
          <span className="inline-block rounded bg-secondary px-1.5 py-0.5 text-[10px] font-medium text-secondary-foreground">
            {node.kind}
          </span>
        </div>

        {incomingEdges.length > 0 && (
          <div>
            <p className="mb-1 text-xs font-medium text-muted-foreground">
              Incoming ({incomingEdges.length})
            </p>
            <ul className="space-y-1">
              {incomingEdges.map((e) => (
                <li
                  key={`${e.source_id}-${e.derivation_type}`}
                  className="rounded bg-blue-900/30 px-2 py-1 text-xs text-blue-400"
                >
                  <span className="font-medium">{e.derivation_type.replace(/_/g, " ")}</span>
                  {e.expression && (
                    <span className="ml-1 font-mono text-[10px] text-blue-500">{e.expression}</span>
                  )}
                </li>
              ))}
            </ul>
          </div>
        )}

        {outgoingEdges.length > 0 && (
          <div>
            <p className="mb-1 text-xs font-medium text-muted-foreground">
              Outgoing ({outgoingEdges.length})
            </p>
            <ul className="space-y-1">
              {outgoingEdges.map((e) => (
                <li
                  key={`${e.target_id}-${e.derivation_type}`}
                  className="rounded bg-emerald-900/30 px-2 py-1 text-xs text-emerald-400"
                >
                  <span className="font-medium">{e.derivation_type.replace(/_/g, " ")}</span>
                  {e.expression && (
                    <span className="ml-1 font-mono text-[10px] text-emerald-500">
                      {e.expression}
                    </span>
                  )}
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </div>
  );
}
