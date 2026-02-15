import { useState } from "react";
import { useIndexRuns, useTriggerIndexRun } from "../../api/hooks";
import type { IndexRun } from "../../api/types";
import { Button } from "../ui/button";
import { ErrorState } from "../ui/ErrorState";
import { Skeleton } from "../ui/skeleton";
import { IndexRunDetail } from "./IndexRunDetail";
import { IndexRunTimeline } from "./IndexRunTimeline";

interface Props {
  projectSlug: string;
}

export function IndexRunList({ projectSlug }: Props) {
  const { data, isLoading, error, refetch } = useIndexRuns(projectSlug);
  const trigger = useTriggerIndexRun(projectSlug);
  const [selectedRun, setSelectedRun] = useState<IndexRun | null>(null);

  if (isLoading) {
    return (
      <div className="space-y-2">
        <Skeleton className="h-4 w-32" />
        <Skeleton className="h-8 w-full" />
      </div>
    );
  }

  if (error) {
    return (
      <ErrorState
        message={error.message || "Failed to load index runs"}
        onRetry={() => refetch()}
      />
    );
  }

  const runs = data?.index_runs ?? [];

  return (
    <div>
      <div className="mb-3 flex items-center justify-between">
        <h4 className="text-sm font-medium">Index Runs</h4>
        <Button size="sm" onClick={() => trigger.mutate(undefined)} disabled={trigger.isPending}>
          {trigger.isPending ? "Triggering..." : "Trigger Run"}
        </Button>
      </div>

      {runs.length === 0 ? (
        <p className="text-sm text-muted-foreground">No index runs yet.</p>
      ) : (
        <>
          <IndexRunTimeline
            runs={runs}
            onSelect={setSelectedRun}
            selectedId={selectedRun?.id ?? null}
          />
          {selectedRun ? (
            <div className="mt-3">
              <IndexRunDetail run={selectedRun} />
            </div>
          ) : (
            <p className="mt-2 text-xs text-muted-foreground">Click a dot to view run details</p>
          )}
        </>
      )}
    </div>
  );
}
