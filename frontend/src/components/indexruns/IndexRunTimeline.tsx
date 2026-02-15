import type { IndexRun } from "../../api/types";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "../ui/tooltip";

interface Props {
  runs: IndexRun[];
  onSelect: (run: IndexRun) => void;
  selectedId: string | null;
}

const STATUS_DOT: Record<string, string> = {
  completed: "bg-green-500",
  failed: "bg-red-500",
  running: "bg-yellow-500 animate-pulse",
  pending: "bg-gray-300",
  queued: "bg-gray-400",
};

export function IndexRunTimeline({ runs, onSelect, selectedId }: Props) {
  return (
    <TooltipProvider delayDuration={200}>
      <div className="flex items-center gap-1 overflow-x-auto py-2">
        {runs.map((run) => (
          <Tooltip key={run.id}>
            <TooltipTrigger asChild>
              <button
                type="button"
                onClick={() => onSelect(run)}
                className={`h-4 w-4 shrink-0 rounded-full transition-transform hover:scale-125 ${STATUS_DOT[run.status] ?? "bg-gray-300"} ${selectedId === run.id ? "ring-2 ring-primary ring-offset-1" : ""}`}
              />
            </TooltipTrigger>
            <TooltipContent>
              <p className="text-xs font-medium capitalize">{run.status}</p>
              <p className="text-xs text-muted-foreground">
                {new Date(run.created_at).toLocaleString()}
              </p>
              <p className="text-xs text-muted-foreground">
                {run.files_processed} files, {run.symbols_found} symbols
              </p>
            </TooltipContent>
          </Tooltip>
        ))}
      </div>
    </TooltipProvider>
  );
}
