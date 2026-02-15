import { useState } from "react";
import { useProjects } from "../api/hooks";
import { CreateProjectDialog } from "../components/projects/CreateProjectDialog";
import { ProjectCard } from "../components/projects/ProjectCard";
import { ErrorState } from "../components/ui/ErrorState";

export function ProjectList() {
  const [showCreate, setShowCreate] = useState(false);
  const { data, isLoading, error, refetch } = useProjects();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-12">
        <p className="text-gray-500">Loading projects...</p>
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
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h2 className="text-2xl font-bold text-gray-900">Projects</h2>
        <button
          type="button"
          onClick={() => setShowCreate(true)}
          className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
        >
          New Project
        </button>
      </div>

      {projects.length === 0 ? (
        <div className="rounded-lg border-2 border-dashed border-gray-300 p-12 text-center">
          <p className="text-gray-500">
            No projects yet. Create your first project to get started.
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
