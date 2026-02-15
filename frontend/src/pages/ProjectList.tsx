import { useState } from "react";
import { useProjects } from "../api/hooks";
import { CreateProjectDialog } from "../components/projects/CreateProjectDialog";
import { ProjectCard } from "../components/projects/ProjectCard";
import { Button } from "../components/ui/button";
import { ErrorState } from "../components/ui/ErrorState";
import { Skeleton } from "../components/ui/skeleton";

export function ProjectList() {
  const [showCreate, setShowCreate] = useState(false);
  const { data, isLoading, error, refetch } = useProjects();

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <Skeleton className="h-8 w-32" />
          <Skeleton className="h-9 w-28" />
        </div>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={`skel-${i.toString()}`} className="h-40" />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <ErrorState message={error.message || "Failed to load projects"} onRetry={() => refetch()} />
    );
  }

  const projects = data?.projects ?? [];

  return (
    <div className="animate-fade-in">
      <div className="mb-6 flex items-center justify-between">
        <h2 className="text-2xl font-bold text-foreground">Projects</h2>
        <Button onClick={() => setShowCreate(true)}>New Project</Button>
      </div>

      {projects.length === 0 ? (
        <div className="rounded-lg border border-dashed border-border p-12 text-center">
          <div className="mx-auto mb-4 flex items-center justify-center gap-3">
            <span className="inline-block h-3 w-3 rounded-full bg-primary/60 animate-pulse-glow" />
            <span className="inline-block h-2 w-8 rounded-full bg-primary/20" />
            <span
              className="inline-block h-3 w-3 rounded-full bg-primary/40 animate-pulse-glow"
              style={{ animationDelay: "0.5s" }}
            />
            <span className="inline-block h-2 w-8 rounded-full bg-primary/20" />
            <span
              className="inline-block h-3 w-3 rounded-full bg-primary/20 animate-pulse-glow"
              style={{ animationDelay: "1s" }}
            />
          </div>
          <p className="text-sm text-muted-foreground">
            No projects yet. Create your first project to start mapping your codebase.
          </p>
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {projects.map((project) => (
            <ProjectCard key={project.id} project={project} />
          ))}
        </div>
      )}

      <CreateProjectDialog open={showCreate} onClose={() => setShowCreate(false)} />
    </div>
  );
}
