import type { ColumnLineageNode, ColumnLineageEdge } from "../../api/types";

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
    <div className="w-72 overflow-y-auto border-l border-gray-200 bg-white p-4">
      <div className="mb-3 flex items-center justify-between">
        <h4 className="text-sm font-semibold text-gray-900">Column Details</h4>
        <button
          type="button"
          onClick={onClose}
          className="text-gray-400 hover:text-gray-600"
        >
          &times;
        </button>
      </div>

      <div className="space-y-3">
        <div>
          <p className="text-xs font-medium text-gray-500">Name</p>
          <p className="text-sm font-medium text-gray-900">{node.name}</p>
        </div>

        <div>
          <p className="text-xs font-medium text-gray-500">Table</p>
          <p className="text-sm text-gray-700">{node.table_name || "N/A"}</p>
        </div>

        <div>
          <p className="text-xs font-medium text-gray-500">Qualified Name</p>
          <p className="break-all font-mono text-xs text-gray-600">
            {node.qualified_name}
          </p>
        </div>

        <div>
          <p className="text-xs font-medium text-gray-500">Kind</p>
          <span className="inline-block rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-medium text-gray-600">
            {node.kind}
          </span>
        </div>

        {incomingEdges.length > 0 && (
          <div>
            <p className="mb-1 text-xs font-medium text-gray-500">
              Incoming ({incomingEdges.length})
            </p>
            <ul className="space-y-1">
              {incomingEdges.map((e, i) => (
                <li
                  key={i}
                  className="rounded bg-blue-50 px-2 py-1 text-xs text-blue-700"
                >
                  <span className="font-medium">
                    {e.derivation_type.replace(/_/g, " ")}
                  </span>
                  {e.expression && (
                    <span className="ml-1 font-mono text-[10px] text-blue-500">
                      {e.expression}
                    </span>
                  )}
                </li>
              ))}
            </ul>
          </div>
        )}

        {outgoingEdges.length > 0 && (
          <div>
            <p className="mb-1 text-xs font-medium text-gray-500">
              Outgoing ({outgoingEdges.length})
            </p>
            <ul className="space-y-1">
              {outgoingEdges.map((e, i) => (
                <li
                  key={i}
                  className="rounded bg-green-50 px-2 py-1 text-xs text-green-700"
                >
                  <span className="font-medium">
                    {e.derivation_type.replace(/_/g, " ")}
                  </span>
                  {e.expression && (
                    <span className="ml-1 font-mono text-[10px] text-green-500">
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
