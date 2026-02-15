import { useDeleteSource, useSources } from "../../api/hooks";
import type { Source } from "../../api/types";
import { ErrorState } from "../ui/ErrorState";

interface Props {
  projectSlug: string;
}

export function SourceList({ projectSlug }: Props) {
  const { data, isLoading, error, refetch } = useSources(projectSlug);
  const deleteSource = useDeleteSource(projectSlug);

  if (isLoading) {
    return <p className="text-sm text-gray-500">Loading sources...</p>;
  }

  if (error) {
    return (
      <ErrorState message={error.message || "Failed to load sources"} onRetry={() => refetch()} />
    );
  }

  const sources = data?.sources ?? [];

  if (sources.length === 0) {
    return <p className="text-sm text-gray-500">No sources configured yet.</p>;
  }

  return (
    <ul className="divide-y divide-gray-100">
      {sources.map((source: Source) => (
        <li key={source.id} className="flex items-center justify-between py-3">
          <div>
            <p className="text-sm font-medium text-gray-900">{source.name}</p>
            <p className="text-xs text-gray-500">
              {source.source_type}
              {source.last_synced_at &&
                ` \u00B7 Last synced ${new Date(source.last_synced_at).toLocaleString()}`}
            </p>
          </div>
          <button
            type="button"
            onClick={() => deleteSource.mutate(source.id)}
            className="text-xs text-red-600 hover:text-red-800"
          >
            Remove
          </button>
        </li>
      ))}
    </ul>
  );
}
