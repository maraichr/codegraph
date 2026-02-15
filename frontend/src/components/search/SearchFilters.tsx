import { Checkbox } from "../ui/checkbox";
import { Label } from "../ui/label";
import { Separator } from "../ui/separator";

interface Props {
  selectedKinds: string[];
  onKindsChange: (kinds: string[]) => void;
  selectedLanguages: string[];
  onLanguagesChange: (languages: string[]) => void;
}

const COMMON_KINDS = [
  "function",
  "class",
  "method",
  "interface",
  "table",
  "column",
  "procedure",
  "view",
  "type",
  "variable",
];

const COMMON_LANGUAGES = ["go", "sql", "typescript", "javascript", "python", "java", "rust", "c#"];

export function SearchFilters({
  selectedKinds,
  onKindsChange,
  selectedLanguages,
  onLanguagesChange,
}: Props) {
  const toggleKind = (kind: string) => {
    onKindsChange(
      selectedKinds.includes(kind)
        ? selectedKinds.filter((k) => k !== kind)
        : [...selectedKinds, kind],
    );
  };

  const toggleLanguage = (lang: string) => {
    onLanguagesChange(
      selectedLanguages.includes(lang)
        ? selectedLanguages.filter((l) => l !== lang)
        : [...selectedLanguages, lang],
    );
  };

  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-sm font-medium">Kind</h3>
        <div className="mt-2 space-y-2">
          {COMMON_KINDS.map((kind) => (
            <div key={kind} className="flex items-center gap-2">
              <Checkbox
                id={`kind-${kind}`}
                checked={selectedKinds.includes(kind)}
                onCheckedChange={() => toggleKind(kind)}
              />
              <Label htmlFor={`kind-${kind}`} className="text-sm capitalize">
                {kind}
              </Label>
            </div>
          ))}
        </div>
      </div>
      <Separator />
      <div>
        <h3 className="text-sm font-medium">Language</h3>
        <div className="mt-2 space-y-2">
          {COMMON_LANGUAGES.map((lang) => (
            <div key={lang} className="flex items-center gap-2">
              <Checkbox
                id={`lang-${lang}`}
                checked={selectedLanguages.includes(lang)}
                onCheckedChange={() => toggleLanguage(lang)}
              />
              <Label htmlFor={`lang-${lang}`} className="text-sm">
                {lang}
              </Label>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
