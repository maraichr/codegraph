interface Props {
  direction: string;
  onDirectionChange: (d: string) => void;
  depth: number;
  onDepthChange: (d: number) => void;
  derivationTypes: string[];
  selectedDerivations: string[];
  onDerivationsChange: (d: string[]) => void;
}

const DIRECTIONS = [
  { value: "upstream", label: "Upstream" },
  { value: "downstream", label: "Downstream" },
  { value: "both", label: "Both" },
];

export function LineageControls({
  direction,
  onDirectionChange,
  depth,
  onDepthChange,
  derivationTypes,
  selectedDerivations,
  onDerivationsChange,
}: Props) {
  const toggleDerivation = (dt: string) => {
    if (selectedDerivations.includes(dt)) {
      onDerivationsChange(selectedDerivations.filter((d) => d !== dt));
    } else {
      onDerivationsChange([...selectedDerivations, dt]);
    }
  };

  return (
    <div className="flex flex-wrap items-center gap-4 border-b border-gray-200 bg-gray-50 px-4 py-2">
      <div className="flex items-center gap-2">
        <span className="text-xs font-medium text-gray-600">Direction:</span>
        <div className="flex rounded-md border border-gray-300">
          {DIRECTIONS.map((d) => (
            <button
              key={d.value}
              type="button"
              onClick={() => onDirectionChange(d.value)}
              className={`px-2.5 py-1 text-xs font-medium ${
                direction === d.value
                  ? "bg-blue-600 text-white"
                  : "bg-white text-gray-700 hover:bg-gray-50"
              } first:rounded-l-md last:rounded-r-md`}
            >
              {d.label}
            </button>
          ))}
        </div>
      </div>

      <div className="flex items-center gap-2">
        <span className="text-xs font-medium text-gray-600">Depth:</span>
        <input
          type="range"
          min={1}
          max={10}
          value={depth}
          onChange={(e) => onDepthChange(parseInt(e.target.value, 10))}
          className="h-1.5 w-24"
        />
        <span className="w-4 text-xs text-gray-500">{depth}</span>
      </div>

      {derivationTypes.length > 0 && (
        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-gray-600">Filter:</span>
          <div className="flex flex-wrap gap-1">
            {derivationTypes.map((dt) => (
              <button
                key={dt}
                type="button"
                onClick={() => toggleDerivation(dt)}
                className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                  selectedDerivations.includes(dt)
                    ? "bg-blue-100 text-blue-700"
                    : "bg-gray-100 text-gray-500"
                }`}
              >
                {dt.replace(/_/g, " ")}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
