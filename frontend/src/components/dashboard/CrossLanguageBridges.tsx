import type { CrossLanguageBridge } from "../../api/types";
import { Badge } from "../ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";

interface Props {
  bridges: CrossLanguageBridge[];
}

export function CrossLanguageBridges({ bridges }: Props) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Cross-Language Bridges</CardTitle>
      </CardHeader>
      <CardContent>
        {bridges.length === 0 ? (
          <p className="text-xs text-muted-foreground">No cross-language edges detected</p>
        ) : (
          <div className="space-y-2">
            {bridges.map((b) => (
              <div
                key={`${b.source_language}-${b.target_language}-${b.edge_type}`}
                className="flex items-center gap-2 text-xs"
              >
                <Badge variant="outline">{b.source_language}</Badge>
                <span className="text-muted-foreground">&rarr;</span>
                <Badge variant="outline">{b.target_language}</Badge>
                <span className="text-muted-foreground">{b.edge_type.replace(/_/g, " ")}</span>
                <span className="ml-auto font-medium">{b.edge_count} edges</span>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
