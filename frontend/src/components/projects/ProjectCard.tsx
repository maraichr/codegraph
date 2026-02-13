import { Link } from "react-router";
import type { Project } from "../../api/types";

interface Props {
  project: Project;
}

export function ProjectCard({ project }: Props) {
  return (
    <Link
      to={`/projects/${project.slug}`}
      className="block rounded-lg border border-gray-200 bg-white p-6 shadow-sm transition hover:shadow-md"
    >
      <h3 className="font-semibold text-gray-900">{project.name}</h3>
      <p className="mt-1 text-sm font-mono text-gray-500">{project.slug}</p>
      {project.description && (
        <p className="mt-2 text-sm text-gray-600 line-clamp-2">
          {project.description}
        </p>
      )}
      <p className="mt-3 text-xs text-gray-400">
        Created {new Date(project.created_at).toLocaleDateString()}
      </p>
    </Link>
  );
}
