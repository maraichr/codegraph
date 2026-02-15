import type { KindCount } from "../../api/types";
import { Badge } from "../ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import { Skeleton } from "../ui/skeleton";

interface Props {
  kinds: KindCount[] | undefined;
  isLoading: boolean;
}

export function KindBreakdown({ kinds, isLoading }: Props) {
  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Symbol Kinds</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-3/4" />
          </div>
        </CardContent>
      </Card>
    );
  }

  if (!kinds?.length) return null;

  const max = Math.max(...kinds.map((k) => k.cnt));

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Symbol Kinds</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        {kinds.map((kind) => (
          <div key={kind.kind} className="flex items-center gap-3">
            <Badge variant="outline" className="w-24 justify-center text-xs">
              {kind.kind}
            </Badge>
            <div className="flex-1">
              <div className="h-2 rounded-full bg-muted">
                <div
                  className="h-2 rounded-full bg-primary"
                  style={{ width: `${(kind.cnt / max) * 100}%` }}
                />
              </div>
            </div>
            <span className="w-12 text-right text-xs text-muted-foreground">{kind.cnt}</span>
          </div>
        ))}
      </CardContent>
    </Card>
  );
}
