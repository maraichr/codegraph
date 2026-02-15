import { Label } from "../ui/label";
import { Slider } from "../ui/slider";

interface Props {
  topK: number;
  onTopKChange: (value: number) => void;
}

export function SemanticControls({ topK, onTopKChange }: Props) {
  return (
    <div className="rounded-md border bg-muted/50 p-4">
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <Label>Results (top_k)</Label>
          <span className="text-sm text-muted-foreground">{topK}</span>
        </div>
        <Slider value={[topK]} onValueChange={([v]) => onTopKChange(v)} min={5} max={50} step={5} />
      </div>
    </div>
  );
}
