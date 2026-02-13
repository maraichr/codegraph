import type { LineageGraph, LineageNode } from "../../api/types";

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
    <div className="w-72 overflow-y-auto border-l border-gray-200 bg-white p-4">
      <div className="mb-3 flex items-center justify-between">
        <h4 className="text-sm font-semibold text-gray-900">Symbol Detail</h4>
        <button
          type="button"
          onClick={onClose}
          className="text-gray-400 hover:text-gray-600"
        >
          x
        </button>
      </div>

      <dl className="space-y-2 text-xs">
        <div>
          <dt className="font-medium text-gray-500">Name</dt>
          <dd className="font-mono text-gray-900">{node.Name}</dd>
        </div>
        <div>
          <dt className="font-medium text-gray-500">Qualified Name</dt>
          <dd className="font-mono text-gray-900">{node.QualifiedName}</dd>
        </div>
        <div>
          <dt className="font-medium text-gray-500">Kind</dt>
          <dd>
            <span className="inline-block rounded bg-gray-100 px-1.5 py-0.5 font-mono">
              {node.Kind}
            </span>
          </dd>
        </div>
        <div>
          <dt className="font-medium text-gray-500">Language</dt>
          <dd className="text-gray-900">{node.Language}</dd>
        </div>
      </dl>

      {incoming.length > 0 && (
        <div className="mt-4">
          <h5 className="text-xs font-medium text-gray-500">
            Incoming ({incoming.length})
          </h5>
          <ul className="mt-1 space-y-1">
            {incoming.map((e, i) => (
              <li key={i} className="text-xs text-gray-700">
                <span className="font-mono">{getNodeName(e.SourceID)}</span>
                <span className="mx-1 text-gray-400">{e.EdgeType}</span>
              </li>
            ))}
          </ul>
        </div>
      )}

      {outgoing.length > 0 && (
        <div className="mt-4">
          <h5 className="text-xs font-medium text-gray-500">
            Outgoing ({outgoing.length})
          </h5>
          <ul className="mt-1 space-y-1">
            {outgoing.map((e, i) => (
              <li key={i} className="text-xs text-gray-700">
                <span className="mx-1 text-gray-400">{e.EdgeType}</span>
                <span className="font-mono">{getNodeName(e.TargetID)}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
