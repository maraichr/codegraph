import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "./client";
import type {
  ColumnLineageGraph,
  CreateProjectInput,
  CreateSourceInput,
  ImpactAnalysisResult,
  IndexRunListResponse,
  LineageGraph,
  Project,
  ProjectListResponse,
  SemanticSearchResponse,
  Source,
  SourceListResponse,
  SymbolSearchResponse,
  UploadResponse,
} from "./types";

// Projects

export function useProjects() {
  return useQuery({
    queryKey: ["projects"],
    queryFn: () => apiClient.get<ProjectListResponse>("/api/v1/projects"),
  });
}

export function useProject(slug: string) {
  return useQuery({
    queryKey: ["project", slug],
    queryFn: () => apiClient.get<Project>(`/api/v1/projects/${slug}`),
    enabled: !!slug,
  });
}

export function useCreateProject() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateProjectInput) =>
      apiClient.post<Project>("/api/v1/projects", input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects"] }),
  });
}

export function useUpdateProject(slug: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { name?: string; description?: string }) =>
      apiClient.put<Project>(`/api/v1/projects/${slug}`, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["projects"] });
      qc.invalidateQueries({ queryKey: ["project", slug] });
    },
  });
}

export function useDeleteProject() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (slug: string) => apiClient.delete(`/api/v1/projects/${slug}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects"] }),
  });
}

// Sources

export function useSources(projectSlug: string) {
  return useQuery({
    queryKey: ["sources", projectSlug],
    queryFn: () =>
      apiClient.get<SourceListResponse>(
        `/api/v1/projects/${projectSlug}/sources`,
      ),
    enabled: !!projectSlug,
  });
}

export function useCreateSource(projectSlug: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateSourceInput) =>
      apiClient.post<Source>(
        `/api/v1/projects/${projectSlug}/sources`,
        input,
      ),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: ["sources", projectSlug] }),
  });
}

export function useDeleteSource(projectSlug: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (sourceId: string) =>
      apiClient.delete(
        `/api/v1/projects/${projectSlug}/sources/${sourceId}`,
      ),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: ["sources", projectSlug] }),
  });
}

export function useUpload(projectSlug: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (file: File) =>
      apiClient.upload<UploadResponse>(
        `/api/v1/projects/${projectSlug}/upload`,
        file,
      ),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["sources", projectSlug] });
      qc.invalidateQueries({ queryKey: ["indexRuns", projectSlug] });
    },
  });
}

// Index Runs

export function useIndexRuns(projectSlug: string) {
  return useQuery({
    queryKey: ["indexRuns", projectSlug],
    queryFn: () =>
      apiClient.get<IndexRunListResponse>(
        `/api/v1/projects/${projectSlug}/index-runs`,
      ),
    enabled: !!projectSlug,
    refetchInterval: (query) => {
      const data = query.state.data;
      if (!data) return false;
      const hasRunning = data.index_runs.some(
        (r) => r.status === "pending" || r.status === "running",
      );
      return hasRunning ? 5000 : false;
    },
  });
}

export function useTriggerIndexRun(projectSlug: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (sourceId: string | undefined = undefined) =>
      apiClient.post(
        `/api/v1/projects/${projectSlug}/index-runs${sourceId ? `?source_id=${sourceId}` : ""}`,
        {},
      ),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: ["indexRuns", projectSlug] }),
  });
}

// Symbols

export function useSymbolSearch(
  projectSlug: string,
  query: string,
  kinds?: string[],
) {
  const kindParam = kinds?.length ? `&kind=${kinds.join(",")}` : "";
  return useQuery({
    queryKey: ["symbols", projectSlug, query, kinds],
    queryFn: () =>
      apiClient.get<SymbolSearchResponse>(
        `/api/v1/projects/${projectSlug}/symbols?q=${encodeURIComponent(query)}${kindParam}`,
      ),
    enabled: !!projectSlug && query.length >= 2,
  });
}

export function useSymbolLineage(
  symbolId: string,
  direction: string,
  depth: number,
) {
  return useQuery({
    queryKey: ["lineage", symbolId, direction, depth],
    queryFn: () =>
      apiClient.get<LineageGraph>(
        `/api/v1/symbols/${symbolId}/lineage?direction=${direction}&max_depth=${depth}`,
      ),
    enabled: !!symbolId,
  });
}

export function useSemanticSearch(projectSlug: string) {
  return useMutation({
    mutationFn: (params: {
      query: string;
      kinds?: string[];
      top_k?: number;
    }) =>
      apiClient.post<SemanticSearchResponse>(
        `/api/v1/projects/${projectSlug}/search/semantic`,
        params,
      ),
  });
}

// Column Lineage

export function useColumnLineage(
  columnId: string,
  direction: string,
  depth: number,
) {
  return useQuery({
    queryKey: ["columnLineage", columnId, direction, depth],
    queryFn: () =>
      apiClient.get<ColumnLineageGraph>(
        `/api/v1/symbols/${columnId}/column-lineage?direction=${direction}&max_depth=${depth}`,
      ),
    enabled: !!columnId,
  });
}

// Impact Analysis

export function useImpactAnalysis(
  symbolId: string,
  changeType: string,
  maxDepth: number,
) {
  return useQuery({
    queryKey: ["impact", symbolId, changeType, maxDepth],
    queryFn: () =>
      apiClient.get<ImpactAnalysisResult>(
        `/api/v1/symbols/${symbolId}/impact?change_type=${changeType}&max_depth=${maxDepth}`,
      ),
    enabled: !!symbolId,
  });
}
