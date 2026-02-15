import type { OracleTruncationData } from "../../../api/types";

interface Props {
  data: unknown;
}

export function TruncationBlock({ data }: Props) {
  const d = data as OracleTruncationData;

  return (
    <p className="text-[10px] text-muted-foreground text-center py-1">
      Showing {d.shown} of {d.total} results
    </p>
  );
}
