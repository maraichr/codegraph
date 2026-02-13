interface Props {
  kinds: string[];
  selectedKinds: string[];
  onKindsChange: (kinds: string[]) => void;
  depth: number;
  onDepthChange: (depth: number) => void;
  direction: string;
  onDirectionChange: (direction: string) => void;
}

const ALL_KINDS = [
  "table",
  "view",
  "procedure",
  "function",
  "trigger",
  "class",
  "interface",
  "method",
  "column",
];

export function GraphFilters({
  selectedKinds,
  onKindsChange,
  depth,
  onDepthChange,
  direction,
  onDirectionChange,
}: Props) {
  const toggleKind = (kind: string) => {
    if (selectedKinds.includes(kind)) {
      onKindsChange(selectedKinds.filter((k) => k !== kind));
    } else {
      onKindsChange([...selectedKinds, kind]);
    }
  };

  return (
    <div className="space-y-3 border-b border-gray-200 bg-white px-4 py-3">
      <div className="flex items-center gap-2">
        <span className="text-xs font-medium text-gray-500">Direction:</span>
        {["upstream", "downstream", "both"].map((d) => (
          <button
            key={d}
            type="button"
            onClick={() => onDirectionChange(d)}
            className={`rounded px-2 py-0.5 text-xs ${
              direction === d
                ? "bg-blue-100 text-blue-700"
                : "text-gray-600 hover:bg-gray-100"
            }`}
          >
            {d}
          </button>
        ))}
        <span className="ml-4 text-xs font-medium text-gray-500">Depth:</span>
        <select
          value={depth}
          onChange={(e) => onDepthChange(Number(e.target.value))}
          className="rounded border border-gray-300 px-1 py-0.5 text-xs"
        >
          {[1, 2, 3, 4, 5].map((d) => (
            <option key={d} value={d}>
              {d}
            </option>
          ))}
        </select>
      </div>
      <div className="flex flex-wrap gap-1">
        {ALL_KINDS.map((kind) => (
          <button
            key={kind}
            type="button"
            onClick={() => toggleKind(kind)}
            className={`rounded px-1.5 py-0.5 text-xs ${
              selectedKinds.length === 0 || selectedKinds.includes(kind)
                ? "bg-gray-200 text-gray-800"
                : "bg-gray-50 text-gray-400"
            }`}
          >
            {kind}
          </button>
        ))}
      </div>
    </div>
  );
}
