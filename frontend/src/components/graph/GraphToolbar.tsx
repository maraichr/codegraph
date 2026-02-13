interface Props {
  layout: string;
  onLayoutChange: (layout: string) => void;
  onFit: () => void;
}

export function GraphToolbar({ layout, onLayoutChange, onFit }: Props) {
  return (
    <div className="flex items-center gap-2 border-b border-gray-200 bg-white px-4 py-2">
      <span className="text-xs font-medium text-gray-500">Layout:</span>
      {["dagre", "cose", "breadthfirst"].map((l) => (
        <button
          key={l}
          type="button"
          onClick={() => onLayoutChange(l)}
          className={`rounded px-2 py-1 text-xs font-medium ${
            layout === l
              ? "bg-blue-100 text-blue-700"
              : "text-gray-600 hover:bg-gray-100"
          }`}
        >
          {l}
        </button>
      ))}
      <div className="ml-auto">
        <button
          type="button"
          onClick={onFit}
          className="rounded px-2 py-1 text-xs font-medium text-gray-600 hover:bg-gray-100"
        >
          Fit
        </button>
      </div>
    </div>
  );
}
