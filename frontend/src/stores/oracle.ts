import { create } from "zustand";
import type { OracleBlock, OracleHint, OracleResponse, OracleResponseMeta } from "../api/types";

export interface OracleMessage {
  id: string;
  role: "user" | "oracle";
  content: string;
  blocks?: OracleBlock[];
  hints?: OracleHint[];
  tool?: string;
  meta?: OracleResponseMeta;
  timestamp: number;
  isLoading?: boolean;
}

interface OracleState {
  isOpen: boolean;
  messages: OracleMessage[];
  sessionId: string | null;
  projectSlug: string | null;

  toggle: () => void;
  open: (slug: string) => void;
  close: () => void;
  addUserMessage: (text: string) => string;
  addLoadingMessage: () => string;
  setOracleResponse: (msgId: string, resp: OracleResponse) => void;
  setOracleError: (msgId: string, error: string) => void;
  clearSession: () => void;
}

let nextId = 0;

export const useOracleStore = create<OracleState>((set, get) => ({
  isOpen: false,
  messages: [],
  sessionId: null,
  projectSlug: null,

  toggle: () => {
    set((s) => ({ isOpen: !s.isOpen }));
  },

  open: (slug: string) => {
    const state = get();
    if (state.projectSlug !== slug) {
      set({ isOpen: true, projectSlug: slug, messages: [], sessionId: null });
    } else {
      set({ isOpen: true });
    }
  },

  close: () => set({ isOpen: false }),

  addUserMessage: (text: string) => {
    const id = `msg-${++nextId}`;
    set((s) => ({
      messages: [
        ...s.messages,
        { id, role: "user", content: text, timestamp: Date.now() },
      ],
    }));
    return id;
  },

  addLoadingMessage: () => {
    const id = `msg-${++nextId}`;
    set((s) => ({
      messages: [
        ...s.messages,
        { id, role: "oracle", content: "", timestamp: Date.now(), isLoading: true },
      ],
    }));
    return id;
  },

  setOracleResponse: (msgId: string, resp: OracleResponse) => {
    set((s) => ({
      sessionId: resp.session_id,
      messages: s.messages.map((m) =>
        m.id === msgId
          ? {
              ...m,
              content: "",
              blocks: resp.blocks,
              hints: resp.hints,
              tool: resp.tool,
              meta: resp.meta,
              isLoading: false,
            }
          : m,
      ),
    }));
  },

  setOracleError: (msgId: string, error: string) => {
    set((s) => ({
      messages: s.messages.map((m) =>
        m.id === msgId
          ? { ...m, content: error, isLoading: false }
          : m,
      ),
    }));
  },

  clearSession: () => set({ messages: [], sessionId: null }),
}));
