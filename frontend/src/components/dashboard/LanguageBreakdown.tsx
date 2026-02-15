import type { LanguageCount } from "../../api/types";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import { Skeleton } from "../ui/skeleton";

interface Props {
  languages: LanguageCount[] | undefined;
  isLoading: boolean;
}

const COLORS = [
  "bg-cyan-500",
  "bg-emerald-500",
  "bg-amber-500",
  "bg-violet-500",
  "bg-rose-500",
  "bg-orange-500",
  "bg-teal-500",
  "bg-fuchsia-500",
];

export function LanguageBreakdown({ languages, isLoading }: Props) {
  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Languages</CardTitle>
        </CardHeader>
        <CardContent>
          <Skeleton className="h-6 w-full" />
          <div className="mt-3 space-y-2">
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-4 w-1/2" />
          </div>
        </CardContent>
      </Card>
    );
  }

  if (!languages?.length) return null;

  const total = languages.reduce((s, l) => s + l.cnt, 0);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Languages</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex h-3 overflow-hidden rounded-full bg-muted">
          {languages.map((lang, i) => (
            <div
              key={lang.language}
              className={`${COLORS[i % COLORS.length]} transition-all`}
              style={{ width: `${(lang.cnt / total) * 100}%` }}
              title={`${lang.language}: ${lang.cnt}`}
            />
          ))}
        </div>
        <div className="mt-3 space-y-1">
          {languages.map((lang, i) => (
            <div key={lang.language} className="flex items-center justify-between text-sm">
              <div className="flex items-center gap-2">
                <div className={`h-2.5 w-2.5 rounded-full ${COLORS[i % COLORS.length]}`} />
                <span>{lang.language}</span>
              </div>
              <span className="text-muted-foreground">
                {lang.cnt} ({((lang.cnt / total) * 100).toFixed(1)}%)
              </span>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
