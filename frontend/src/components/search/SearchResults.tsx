import { Link } from "react-router";
import type { GlobalSymbol } from "../../api/types";
import { Badge } from "../ui/badge";
import { Skeleton } from "../ui/skeleton";

interface Props {
  results: GlobalSymbol[] | undefined;
  isLoading: boolean;
  count: number | undefined;
}

export function SearchResults({ results, isLoading, count }: Props) {
  if (isLoading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={`skel-${i.toString()}`} className="h-16 w-full" />
        ))}
      </div>
    );
  }

  if (!results?.length) {
    return <p className="text-sm text-muted-foreground">No results found.</p>;
  }

  return (
    <div className="space-y-2">
      <p className="text-sm text-muted-foreground">{count} results</p>
      {results.map((sym) => (
        <Link
          key={sym.id}
          to={`/projects/${sym.project_slug}/graph`}
          className="block rounded-md border p-3 transition hover:bg-accent"
        >
          <div className="flex items-start justify-between">
            <div className="min-w-0 flex-1">
              <p className="truncate font-medium">{sym.name}</p>
              <p className="truncate text-sm text-muted-foreground">{sym.qualified_name}</p>
            </div>
            <div className="flex gap-1.5">
              <Badge variant="outline" className="text-xs">
                {sym.kind}
              </Badge>
              <Badge variant="secondary" className="text-xs">
                {sym.language}
              </Badge>
            </div>
          </div>
          <p className="mt-1 text-xs text-muted-foreground">
            {sym.project_slug} &middot; lines {sym.start_line}–{sym.end_line}
          </p>
        </Link>
      ))}
    </div>
  );
}
