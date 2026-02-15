import { useState } from "react";
import { useGlobalSymbolSearch } from "../api/hooks";
import { SearchBar } from "../components/search/SearchBar";
import { SearchFilters } from "../components/search/SearchFilters";
import { SearchResults } from "../components/search/SearchResults";
import { SemanticControls } from "../components/search/SemanticControls";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";

export function Search() {
  const [query, setQuery] = useState("");
  const [mode, setMode] = useState<"text" | "semantic">("text");
  const [kinds, setKinds] = useState<string[]>([]);
  const [languages, setLanguages] = useState<string[]>([]);
  const [topK, setTopK] = useState(20);

  const searchResults = useGlobalSymbolSearch(
    query,
    kinds.length > 0 ? kinds : undefined,
    languages.length > 0 ? languages : undefined,
    mode === "text" ? topK : undefined,
  );

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold">Search</h2>
        <p className="mt-1 text-sm text-muted-foreground">Find symbols across all projects.</p>
      </div>

      <SearchBar query={query} onQueryChange={setQuery} mode={mode} onModeChange={setMode} />

      <div className="grid gap-6 lg:grid-cols-[240px_1fr]">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Filters</CardTitle>
          </CardHeader>
          <CardContent>
            {mode === "semantic" && (
              <div className="mb-4">
                <SemanticControls topK={topK} onTopKChange={setTopK} />
              </div>
            )}
            <SearchFilters
              selectedKinds={kinds}
              onKindsChange={setKinds}
              selectedLanguages={languages}
              onLanguagesChange={setLanguages}
            />
          </CardContent>
        </Card>

        <div>
          {query.length < 2 ? (
            <p className="text-sm text-muted-foreground">Type at least 2 characters to search.</p>
          ) : (
            <SearchResults
              results={searchResults.data?.symbols}
              isLoading={searchResults.isLoading}
              count={searchResults.data?.count}
            />
          )}
        </div>
      </div>
    </div>
  );
}
