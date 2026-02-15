import type { OracleTableData } from "../../../api/types";

interface Props {
  data: unknown;
}

export function TableBlock({ data }: Props) {
  const d = data as OracleTableData;
  if (!d.headers?.length && !d.rows?.length) return null;

  return (
    <div className="overflow-x-auto rounded-md border border-border/50">
      <table className="w-full text-xs">
        {d.headers?.length > 0 && (
          <thead>
            <tr className="border-b border-border/50 bg-muted/30">
              {d.headers.map((h, i) => (
                <th key={i} className="px-2.5 py-1.5 text-left font-medium text-muted-foreground">
                  {h}
                </th>
              ))}
            </tr>
          </thead>
        )}
        <tbody>
          {d.rows?.map((row, i) => (
            <tr key={i} className="border-b border-border/30 last:border-0">
              {row.map((cell, j) => (
                <td key={j} className="px-2.5 py-1.5 text-foreground">
                  {cell}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
