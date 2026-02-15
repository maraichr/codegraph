import type { SourceSymbolStats } from "../../api/types";
import { Badge } from "../ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import { Skeleton } from "../ui/skeleton";

interface Props {
  sources: SourceSymbolStats[] | undefined;
  isLoading: boolean;
}

export function SourceHealth({ sources, isLoading }: Props) {
  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Source Stats</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
          </div>
        </CardContent>
      </Card>
    );
  }

  if (!sources?.length) return null;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Source Stats</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {sources.map((src) => (
          <div key={src.source_id} className="rounded-md border p-3">
            <p className="text-xs font-mono text-muted-foreground">{src.source_id}</p>
            <div className="mt-2 flex flex-wrap gap-3 text-sm">
              <span>
                <strong>{src.symbol_count}</strong> symbols
              </span>
              <span>
                <strong>{src.file_count}</strong> files
              </span>
              <span>
                <strong>{src.language_count}</strong> languages
              </span>
            </div>
            {src.languages?.length > 0 && (
              <div className="mt-2 flex flex-wrap gap-1">
                {src.languages.map((lang) => (
                  <Badge key={lang} variant="secondary" className="text-xs">
                    {lang}
                  </Badge>
                ))}
              </div>
            )}
          </div>
        ))}
      </CardContent>
    </Card>
  );
}
