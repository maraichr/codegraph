import { Button } from "./button";
import { Card, CardContent } from "./card";

interface Props {
  message: string;
  onRetry?: () => void;
}

export function ErrorState({ message, onRetry }: Props) {
  return (
    <Card className="border-destructive/50 bg-destructive/5">
      <CardContent className="p-6 text-center">
        <p className="text-sm text-destructive">{message}</p>
        {onRetry && (
          <Button variant="destructive" size="sm" onClick={onRetry} className="mt-3">
            Retry
          </Button>
        )}
      </CardContent>
    </Card>
  );
}
