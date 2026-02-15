import { useState } from "react";
import { useParams } from "react-router";
import {
  useParserCoverage,
  useProject,
  useProjectBridges,
  useProjectKinds,
  useProjectLanguages,
  useProjectLayers,
  useProjectStats,
  useSourceStats,
  useTopSymbolsByInDegree,
  useTopSymbolsByPageRank,
} from "../api/hooks";
import { CrossLanguageBridges } from "../components/dashboard/CrossLanguageBridges";
import { KindBreakdown } from "../components/dashboard/KindBreakdown";
import { LanguageBreakdown } from "../components/dashboard/LanguageBreakdown";
import { LayerDistribution } from "../components/dashboard/LayerDistribution";
import { ParserCoverage } from "../components/dashboard/ParserCoverage";
import { SourceHealth } from "../components/dashboard/SourceHealth";
import { StatsCards } from "../components/dashboard/StatsCards";
import { TopSymbols } from "../components/dashboard/TopSymbols";
import { IndexRunList } from "../components/indexruns/IndexRunList";
import { AddSourceDialog } from "../components/sources/AddSourceDialog";
import { SourceList } from "../components/sources/SourceList";
import { ZipUpload } from "../components/sources/ZipUpload";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { ErrorState } from "../components/ui/ErrorState";
import { Skeleton } from "../components/ui/skeleton";

export function ProjectDashboard() {
  const { slug } = useParams<{ slug: string }>();
  const { data: project, isLoading, error, refetch } = useProject(slug ?? "");
  const [showAddSource, setShowAddSource] = useState(false);

  const stats = useProjectStats(slug ?? "");
  const languages = useProjectLanguages(slug ?? "");
  const kinds = useProjectKinds(slug ?? "");
  const layers = useProjectLayers(slug ?? "");
  const topInDegree = useTopSymbolsByInDegree(slug ?? "");
  const topPageRank = useTopSymbolsByPageRank(slug ?? "");
  const sourceStats = useSourceStats(slug ?? "");
  const coverage = useParserCoverage(slug ?? "");
  const bridges = useProjectBridges(slug ?? "");

  if (!slug) return null;

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-4 w-72" />
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={`skel-${i.toString()}`} className="h-20" />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <ErrorState message={error.message || "Failed to load project"} onRetry={() => refetch()} />
    );
  }

  if (!project) {
    return (
      <div className="flex items-center justify-center p-12">
        <p className="text-destructive">Project not found.</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold">{project.name}</h2>
        {project.description && (
          <p className="mt-1 text-sm text-muted-foreground">{project.description}</p>
        )}
        <p className="mt-1 text-xs font-mono text-muted-foreground">{project.slug}</p>
      </div>

      <StatsCards stats={stats.data} isLoading={stats.isLoading} />

      <div className="grid gap-6 lg:grid-cols-2">
        <LanguageBreakdown languages={languages.data} isLoading={languages.isLoading} />
        <KindBreakdown kinds={kinds.data} isLoading={kinds.isLoading} />
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <LayerDistribution layers={layers.data} isLoading={layers.isLoading} />
        <TopSymbols
          byInDegree={topInDegree.data}
          byPageRank={topPageRank.data}
          isLoading={topInDegree.isLoading || topPageRank.isLoading}
        />
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader className="flex-row items-center justify-between space-y-0">
            <CardTitle className="text-sm">Sources</CardTitle>
            <Button size="sm" onClick={() => setShowAddSource(true)}>
              Add Source
            </Button>
          </CardHeader>
          <CardContent>
            <SourceList projectSlug={slug} />
            <div className="mt-4 border-t pt-4">
              <p className="mb-2 text-xs font-medium text-muted-foreground">Quick Upload</p>
              <ZipUpload projectSlug={slug} />
            </div>
          </CardContent>
        </Card>

        <SourceHealth sources={sourceStats.data} isLoading={sourceStats.isLoading} />
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {coverage.data && <ParserCoverage coverage={coverage.data} />}
        {bridges.data && <CrossLanguageBridges bridges={bridges.data} />}
      </div>

      <Card>
        <CardContent className="p-6">
          <IndexRunList projectSlug={slug} />
        </CardContent>
      </Card>

      <AddSourceDialog
        projectSlug={slug}
        open={showAddSource}
        onClose={() => setShowAddSource(false)}
      />
    </div>
  );
}
