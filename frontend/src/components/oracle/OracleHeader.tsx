import { Sparkles, X, Trash2 } from "lucide-react";
import { useOracleStore } from "../../stores/oracle";

interface Props {
  onClose: () => void;
  hasSession: boolean;
}

export function OracleHeader({ onClose, hasSession }: Props) {
  const clearSession = useOracleStore((s) => s.clearSession);

  return (
    <div className="flex items-center justify-between border-b border-border px-4 py-3">
      <div className="flex items-center gap-2">
        <Sparkles className="h-5 w-5 text-primary" />
        <h2 className="text-sm font-semibold tracking-wide">The Oracle</h2>
        {hasSession && (
          <span className="h-2 w-2 rounded-full bg-emerald-400" title="Session active" />
        )}
      </div>
      <div className="flex items-center gap-1">
        <button
          onClick={clearSession}
          className="rounded p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
          title="Clear conversation"
        >
          <Trash2 className="h-4 w-4" />
        </button>
        <button
          onClick={onClose}
          className="rounded p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
          title="Close (Cmd+K)"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}
