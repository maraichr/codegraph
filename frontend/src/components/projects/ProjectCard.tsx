import { Link } from "react-router";
import type { Project } from "../../api/types";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";

interface Props {
  project: Project;
}

export function ProjectCard({ project }: Props) {
  return (
    <Link to={`/projects/${project.slug}`}>
      <Card className="transition-all hover:border-primary/30 hover:shadow-[0_0_15px_rgba(0,210,210,0.08)]">
        <CardHeader>
          <CardTitle>{project.name}</CardTitle>
          <p className="text-sm font-mono text-muted-foreground">{project.slug}</p>
        </CardHeader>
        <CardContent>
          {project.description && (
            <p className="text-sm text-muted-foreground line-clamp-2">{project.description}</p>
          )}
          <p className="mt-3 text-xs text-muted-foreground">
            Created {new Date(project.created_at).toLocaleDateString()}
          </p>
        </CardContent>
      </Card>
    </Link>
  );
}
