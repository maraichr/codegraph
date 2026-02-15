import type { ParserCoverageRow } from "../../api/types";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import { Progress } from "../ui/progress";

interface Props {
  coverage: ParserCoverageRow[];
}

export function ParserCoverage({ coverage }: Props) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Parser Coverage</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {coverage.map((row) => {
          const pct =
            row.total_files > 0 ? Math.round((row.parsed_files / row.total_files) * 100) : 0;
          return (
            <div key={row.source_id}>
              <div className="mb-1 flex items-center justify-between text-xs">
                <span className="font-mono text-muted-foreground">{row.source_id.slice(0, 8)}</span>
                <span className="font-medium">
                  {row.parsed_files}/{row.total_files} files ({pct}%)
                </span>
              </div>
              <Progress value={pct} />
            </div>
          );
        })}
        {coverage.length === 0 && (
          <p className="text-xs text-muted-foreground">No coverage data available</p>
        )}
      </CardContent>
    </Card>
  );
}
