import { AlertTriangle } from "lucide-react";
import { Button } from "./button";
import { Card, CardContent } from "./card";

interface Props {
  message: string;
  onRetry?: () => void;
}

export function ErrorState({ message, onRetry }: Props) {
  return (
    <Card className="border-destructive/30 bg-destructive/5">
      <CardContent className="flex flex-col items-center gap-3 p-6 text-center">
        <AlertTriangle className="h-8 w-8 text-destructive/70" />
        <p className="text-sm text-destructive">{message}</p>
        {onRetry && (
          <Button variant="destructive" size="sm" onClick={onRetry}>
            Try Again
          </Button>
        )}
      </CardContent>
    </Card>
  );
}
