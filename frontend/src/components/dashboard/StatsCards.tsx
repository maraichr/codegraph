import { Code2, FileCode, GitFork, Languages } from "lucide-react";
import type { ProjectStats } from "../../api/types";
import { Card, CardContent } from "../ui/card";
import { Skeleton } from "../ui/skeleton";

interface Props {
  stats: ProjectStats | undefined;
  isLoading: boolean;
}

const metrics = [
  { key: "total_symbols" as const, label: "Symbols", icon: Code2 },
  { key: "file_count" as const, label: "Files", icon: FileCode },
  { key: "language_count" as const, label: "Languages", icon: Languages },
  { key: "kind_count" as const, label: "Kinds", icon: GitFork },
];

export function StatsCards({ stats, isLoading }: Props) {
  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      {metrics.map((m) => (
        <Card key={m.key}>
          <CardContent className="flex items-center gap-4 p-4">
            <div className="rounded-md bg-primary/10 p-2">
              <m.icon className="h-5 w-5 text-primary" />
            </div>
            <div>
              {isLoading ? (
                <Skeleton className="h-7 w-16" />
              ) : (
                <p className="text-2xl font-bold">{stats?.[m.key]?.toLocaleString() ?? 0}</p>
              )}
              <p className="text-xs text-muted-foreground">{m.label}</p>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
