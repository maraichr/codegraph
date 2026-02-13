import { useState } from "react";
import { Link, useParams } from "react-router";
import { useProject } from "../api/hooks";
import { IndexRunList } from "../components/indexruns/IndexRunList";
import { AddSourceDialog } from "../components/sources/AddSourceDialog";
import { SourceList } from "../components/sources/SourceList";
import { ZipUpload } from "../components/sources/ZipUpload";
import { ErrorState } from "../components/ui/ErrorState";

export function ProjectDashboard() {
  const { slug } = useParams<{ slug: string }>();
  const { data: project, isLoading, error, refetch } = useProject(slug ?? "");
  const [showAddSource, setShowAddSource] = useState(false);

  if (!slug) return null;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-12">
        <p className="text-gray-500">Loading project...</p>
      </div>
    );
  }

  if (error) {
    return (
      <ErrorState
        message={error.message || "Failed to load project"}
        onRetry={() => refetch()}
      />
    );
  }

  if (!project) {
    return (
      <div className="flex items-center justify-center p-12">
        <p className="text-red-500">Project not found.</p>
      </div>
    );
  }

  return (
    <div>
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-gray-900">{project.name}</h2>
        {project.description && (
          <p className="mt-1 text-sm text-gray-600">{project.description}</p>
        )}
        <p className="mt-1 text-xs font-mono text-gray-400">{project.slug}</p>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <div className="rounded-lg border border-gray-200 bg-white p-6">
          <div className="mb-4 flex items-center justify-between">
            <h3 className="font-semibold text-gray-900">Sources</h3>
            <button
              type="button"
              onClick={() => setShowAddSource(true)}
              className="rounded-md bg-blue-600 px-3 py-1 text-xs font-medium text-white hover:bg-blue-700"
            >
              Add Source
            </button>
          </div>
          <SourceList projectSlug={slug} />
          <div className="mt-4 border-t border-gray-100 pt-4">
            <p className="mb-2 text-xs font-medium text-gray-700">
              Quick Upload
            </p>
            <ZipUpload projectSlug={slug} />
          </div>
        </div>

        <div className="rounded-lg border border-gray-200 bg-white p-6">
          <IndexRunList projectSlug={slug} />
        </div>

        <div className="col-span-full rounded-lg border border-gray-200 bg-white p-6">
          <div className="flex items-center justify-between">
            <h3 className="font-semibold text-gray-900">Dependency Graph</h3>
            <Link
              to={`/projects/${slug}/graph`}
              className="rounded-md bg-blue-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-blue-700"
            >
              Open Graph Explorer
            </Link>
          </div>
          <p className="mt-2 text-sm text-gray-500">
            Explore symbol dependencies, lineage, and relationships in an
            interactive graph.
          </p>
        </div>

        <div className="rounded-lg border border-gray-200 bg-white p-6">
          <div className="flex items-center justify-between">
            <h3 className="font-semibold text-gray-900">Column Lineage</h3>
            <Link
              to={`/projects/${slug}/lineage`}
              className="rounded-md bg-purple-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-purple-700"
            >
              Explore Lineage
            </Link>
          </div>
          <p className="mt-2 text-sm text-gray-500">
            Trace column-level data flows across tables, views, and stored
            procedures.
          </p>
        </div>

        <div className="rounded-lg border border-gray-200 bg-white p-6">
          <div className="flex items-center justify-between">
            <h3 className="font-semibold text-gray-900">Impact Analysis</h3>
            <Link
              to={`/projects/${slug}/impact`}
              className="rounded-md bg-orange-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-orange-700"
            >
              Analyze Impact
            </Link>
          </div>
          <p className="mt-2 text-sm text-gray-500">
            Assess the blast radius of modifying, deleting, or renaming a
            symbol.
          </p>
        </div>
      </div>

      <AddSourceDialog
        projectSlug={slug}
        open={showAddSource}
        onClose={() => setShowAddSource(false)}
      />
    </div>
  );
}
