import type { BadgeProps } from "./badge";
import { Badge } from "./badge";

interface Props {
  status: string;
}

const statusVariants: Record<string, BadgeProps["variant"]> = {
  pending: "warning",
  running: "info",
  completed: "success",
  failed: "destructive",
  cancelled: "secondary",
};

export function StatusBadge({ status }: Props) {
  const variant = statusVariants[status] ?? "secondary";
  return <Badge variant={variant}>{status}</Badge>;
}
