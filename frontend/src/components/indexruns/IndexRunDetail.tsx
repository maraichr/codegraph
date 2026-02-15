import { ChevronDown } from "lucide-react";
import { useState } from "react";
import type { IndexRun } from "../../api/types";
import { Badge } from "../ui/badge";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "../ui/collapsible";

interface Props {
  run: IndexRun;
}

const STATUS_VARIANT: Record<string, "success" | "destructive" | "warning" | "secondary"> = {
  completed: "success",
  failed: "destructive",
  running: "warning",
  pending: "secondary",
  queued: "secondary",
};

export function IndexRunDetail({ run }: Props) {
  const [open, setOpen] = useState(true);

  const duration =
    run.started_at && run.completed_at
      ? Math.round(
          (new Date(run.completed_at).getTime() - new Date(run.started_at).getTime()) / 1000,
        )
      : null;

  return (
    <Collapsible open={open} onOpenChange={setOpen} className="rounded-md border bg-card">
      <CollapsibleTrigger className="flex w-full items-center gap-2 px-4 py-3 text-sm">
        <ChevronDown
          className={`h-4 w-4 text-muted-foreground transition-transform ${open ? "" : "-rotate-90"}`}
        />
        <Badge variant={STATUS_VARIANT[run.status] ?? "secondary"} className="capitalize">
          {run.status}
        </Badge>
        <span className="font-mono text-xs text-muted-foreground">{run.id.slice(0, 8)}</span>
        <span className="ml-auto text-xs text-muted-foreground">
          {new Date(run.created_at).toLocaleString()}
        </span>
      </CollapsibleTrigger>
      <CollapsibleContent className="border-t px-4 py-3">
        <dl className="grid grid-cols-2 gap-x-4 gap-y-2 text-xs">
          <div>
            <dt className="font-medium text-muted-foreground">Files Processed</dt>
            <dd>{run.files_processed}</dd>
          </div>
          <div>
            <dt className="font-medium text-muted-foreground">Symbols Found</dt>
            <dd>{run.symbols_found}</dd>
          </div>
          <div>
            <dt className="font-medium text-muted-foreground">Edges Found</dt>
            <dd>{run.edges_found}</dd>
          </div>
          {duration != null && (
            <div>
              <dt className="font-medium text-muted-foreground">Duration</dt>
              <dd>{duration}s</dd>
            </div>
          )}
        </dl>
        {run.error_message && (
          <div className="mt-2 rounded bg-destructive/10 p-2 text-xs text-destructive">
            {run.error_message}
          </div>
        )}
      </CollapsibleContent>
    </Collapsible>
  );
}
