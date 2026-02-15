const KIND_COLORS: Record<string, string> = {
  table: "#3b82f6",
  view: "#8b5cf6",
  procedure: "#10b981",
  function: "#f59e0b",
  trigger: "#ef4444",
  column: "#6b7280",
  class: "#06b6d4",
  interface: "#a855f7",
  method: "#84cc16",
  package: "#f97316",
};

export function GraphLegend() {
  return (
    <div className="flex flex-wrap gap-3 rounded-md border bg-card p-2 text-xs">
      {Object.entries(KIND_COLORS).map(([kind, color]) => (
        <div key={kind} className="flex items-center gap-1.5">
          <span className="inline-block h-3 w-3 rounded-full" style={{ backgroundColor: color }} />
          <span className="capitalize text-muted-foreground">{kind}</span>
        </div>
      ))}
    </div>
  );
}
