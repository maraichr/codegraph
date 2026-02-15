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
    <div className="flex flex-wrap items-center gap-4 border-b border-border bg-muted px-4 py-2">
      <div className="flex items-center gap-2">
        <span className="text-xs font-medium text-muted-foreground">Direction:</span>
        <div className="flex rounded-md border border-border">
          {DIRECTIONS.map((d) => (
            <button
              key={d.value}
              type="button"
              onClick={() => onDirectionChange(d.value)}
              className={`px-2.5 py-1 text-xs font-medium transition-colors ${
                direction === d.value
                  ? "bg-primary text-primary-foreground"
                  : "bg-secondary text-secondary-foreground hover:bg-accent"
              } first:rounded-l-md last:rounded-r-md`}
            >
              {d.label}
            </button>
          ))}
        </div>
      </div>

      <div className="flex items-center gap-2">
        <span className="text-xs font-medium text-muted-foreground">Depth:</span>
        <input
          type="range"
          min={1}
          max={10}
          value={depth}
          onChange={(e) => onDepthChange(parseInt(e.target.value, 10))}
          className="h-1.5 w-24 accent-primary"
        />
        <span className="w-4 text-xs text-muted-foreground">{depth}</span>
      </div>

      {derivationTypes.length > 0 && (
        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-muted-foreground">Filter:</span>
          <div className="flex flex-wrap gap-1">
            {derivationTypes.map((dt) => (
              <button
                key={dt}
                type="button"
                onClick={() => toggleDerivation(dt)}
                className={`rounded-full px-2 py-0.5 text-[10px] font-medium transition-colors ${
                  selectedDerivations.includes(dt)
                    ? "bg-primary/20 text-primary"
                    : "bg-secondary text-muted-foreground"
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
