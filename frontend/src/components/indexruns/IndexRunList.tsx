import { useIndexRuns, useTriggerIndexRun } from "../../api/hooks";
import type { IndexRun } from "../../api/types";
import { ErrorState } from "../ui/ErrorState";
import { StatusBadge } from "../ui/StatusBadge";

interface Props {
  projectSlug: string;
}

export function IndexRunList({ projectSlug }: Props) {
  const { data, isLoading, error, refetch } = useIndexRuns(projectSlug);
  const trigger = useTriggerIndexRun(projectSlug);

  if (isLoading) {
    return <p className="text-sm text-gray-500">Loading index runs...</p>;
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
        <h4 className="text-sm font-medium text-gray-700">Index Runs</h4>
        <button
          type="button"
          onClick={() => trigger.mutate(undefined)}
          disabled={trigger.isPending}
          className="rounded-md bg-blue-600 px-3 py-1 text-xs font-medium text-white hover:bg-blue-700 disabled:opacity-50"
        >
          {trigger.isPending ? "Triggering..." : "Trigger Run"}
        </button>
      </div>

      {runs.length === 0 ? (
        <p className="text-sm text-gray-500">No index runs yet.</p>
      ) : (
        <ul className="divide-y divide-gray-100">
          {runs.map((run: IndexRun) => (
            <li key={run.id} className="py-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <StatusBadge status={run.status} />
                  <span className="text-xs font-mono text-gray-500">
                    {run.id.slice(0, 8)}
                  </span>
                </div>
                <span className="text-xs text-gray-500">
                  {new Date(run.created_at).toLocaleString()}
                </span>
              </div>
              <div className="mt-1 flex gap-4 text-xs text-gray-500">
                <span>{run.files_processed} files</span>
                <span>{run.symbols_found} symbols</span>
                <span>{run.edges_found} edges</span>
              </div>
              {run.error_message && (
                <p className="mt-1 text-xs text-red-600">
                  {run.error_message}
                </p>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
