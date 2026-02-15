import { Sparkles } from "lucide-react";
import { useParams } from "react-router";
import { useOracleStore } from "../../stores/oracle";

export function OracleFAB() {
  const { slug } = useParams<{ slug: string }>();
  const { isOpen, open } = useOracleStore();

  if (isOpen || !slug) return null;

  return (
    <button
      onClick={() => open(slug)}
      className="fixed bottom-6 right-6 z-40 flex h-14 w-14 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-lg transition-all hover:scale-105 hover:shadow-[0_0_20px_rgba(0,210,210,0.3)] active:scale-95"
      title="Ask The Oracle (Cmd+K)"
    >
      <Sparkles className="h-6 w-6" />
    </button>
  );
}
