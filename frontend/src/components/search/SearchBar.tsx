import { Search, Sparkles } from "lucide-react";
import { Button } from "../ui/button";
import { Input } from "../ui/input";

interface Props {
  query: string;
  onQueryChange: (q: string) => void;
  mode: "text" | "semantic";
  onModeChange: (mode: "text" | "semantic") => void;
}

export function SearchBar({ query, onQueryChange, mode, onModeChange }: Props) {
  return (
    <div className="flex gap-2">
      <div className="relative flex-1">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          value={query}
          onChange={(e) => onQueryChange(e.target.value)}
          placeholder="Search symbols across all projects..."
          className="pl-10"
        />
      </div>
      <Button
        variant={mode === "text" ? "default" : "outline"}
        size="sm"
        onClick={() => onModeChange("text")}
      >
        <Search className="h-4 w-4" />
        Text
      </Button>
      <Button
        variant={mode === "semantic" ? "default" : "outline"}
        size="sm"
        onClick={() => onModeChange("semantic")}
      >
        <Sparkles className="h-4 w-4" />
        Semantic
      </Button>
    </div>
  );
}
