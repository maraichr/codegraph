import type { LayerCount } from "../../api/types";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import { Skeleton } from "../ui/skeleton";

interface Props {
  layers: LayerCount[] | undefined;
  isLoading: boolean;
}

const LAYER_COLORS: Record<string, string> = {
  data: "bg-cyan-500",
  business: "bg-emerald-500",
  api: "bg-amber-500",
  infrastructure: "bg-violet-500",
  "cross-cutting": "bg-rose-500",
};

export function LayerDistribution({ layers, isLoading }: Props) {
  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Architecture Layers</CardTitle>
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

  if (!layers?.length) return null;

  const total = layers.reduce((s, l) => s + l.cnt, 0);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Architecture Layers</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {layers.map((layer) => {
          const color = LAYER_COLORS[layer.layer] ?? "bg-secondary";
          const pct = total > 0 ? ((layer.cnt / total) * 100).toFixed(1) : "0";
          return (
            <div key={layer.layer} className="space-y-1">
              <div className="flex items-center justify-between text-sm">
                <span className="capitalize">{layer.layer}</span>
                <span className="text-muted-foreground">
                  {layer.cnt} ({pct}%)
                </span>
              </div>
              <div className="h-2 rounded-full bg-muted">
                <div
                  className={`h-2 rounded-full ${color} transition-all`}
                  style={{ width: `${(layer.cnt / total) * 100}%` }}
                />
              </div>
            </div>
          );
        })}
      </CardContent>
    </Card>
  );
}
