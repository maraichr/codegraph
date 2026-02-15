import { useEffect, useRef } from "react";
import { useParams } from "react-router";
import { useOracleAsk } from "../../api/hooks";
import { useOracleStore } from "../../stores/oracle";
import { OracleHeader } from "./OracleHeader";
import { OracleInput } from "./OracleInput";
import { OracleMessage } from "./OracleMessage";
import { OracleWelcome } from "./OracleWelcome";

export function OraclePanel() {
  const { slug } = useParams<{ slug: string }>();
  const { isOpen, messages, sessionId, close } = useOracleStore();
  const scrollRef = useRef<HTMLDivElement>(null);

  const askMutation = useOracleAsk(slug || "");
  const {
    addUserMessage,
    addLoadingMessage,
    setOracleResponse,
    setOracleError,
  } = useOracleStore();

  // Keyboard shortcut: Cmd/Ctrl+K
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        useOracleStore.getState().toggle();
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);

  // Auto-scroll on new messages
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages]);

  const handleSend = (question: string) => {
    if (!slug) return;
    addUserMessage(question);
    const loadingId = addLoadingMessage();

    askMutation.mutate(
      { question, session_id: sessionId || undefined },
      {
        onSuccess: (resp) => setOracleResponse(loadingId, resp),
        onError: (err) =>
          setOracleError(loadingId, err instanceof Error ? err.message : "Something went wrong"),
      },
    );
  };

  const handleHintClick = (question: string) => {
    handleSend(question);
  };

  if (!isOpen) return null;

  return (
    <div className="fixed right-0 top-0 z-50 flex h-screen w-[420px] max-w-full flex-col border-l border-border bg-background shadow-2xl animate-in slide-in-from-right duration-200">
      <OracleHeader onClose={close} hasSession={!!sessionId} />
      <div ref={scrollRef} className="flex-1 overflow-y-auto p-4 space-y-4">
        {messages.length === 0 ? (
          <OracleWelcome onQuestionClick={handleHintClick} />
        ) : (
          messages.map((msg) => (
            <OracleMessage
              key={msg.id}
              message={msg}
              onHintClick={handleHintClick}
            />
          ))
        )}
      </div>
      <OracleInput onSend={handleSend} isLoading={askMutation.isPending} />
    </div>
  );
}
