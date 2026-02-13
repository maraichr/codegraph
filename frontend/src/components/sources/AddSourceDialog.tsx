import { useState } from "react";
import { useCreateSource } from "../../api/hooks";

interface Props {
  projectSlug: string;
  open: boolean;
  onClose: () => void;
}

export function AddSourceDialog({ projectSlug, open, onClose }: Props) {
  const [name, setName] = useState("");
  const [sourceType, setSourceType] = useState("git");
  const [connectionUri, setConnectionUri] = useState("");
  const createSource = useCreateSource(projectSlug);

  if (!open) return null;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    createSource.mutate(
      {
        name,
        source_type: sourceType,
        connection_uri: connectionUri || undefined,
      },
      {
        onSuccess: () => {
          setName("");
          setSourceType("git");
          setConnectionUri("");
          onClose();
        },
      },
    );
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
        <h3 className="text-lg font-semibold text-gray-900">Add Source</h3>
        <form onSubmit={handleSubmit} className="mt-4 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700">
              Name
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">
              Type
            </label>
            <select
              value={sourceType}
              onChange={(e) => setSourceType(e.target.value)}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            >
              <option value="git">Git Repository</option>
              <option value="upload">File Upload</option>
              <option value="database">Database</option>
              <option value="filesystem">Filesystem</option>
            </select>
          </div>
          {sourceType === "git" && (
            <div>
              <label className="block text-sm font-medium text-gray-700">
                Repository URL
              </label>
              <input
                type="text"
                value={connectionUri}
                onChange={(e) => setConnectionUri(e.target.value)}
                placeholder="https://gitlab.com/group/repo"
                className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
            </div>
          )}
          {createSource.error && (
            <p className="text-sm text-red-600">
              {(createSource.error as Error).message}
            </p>
          )}
          <div className="flex justify-end gap-3">
            <button
              type="button"
              onClick={onClose}
              className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={createSource.isPending}
              className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
            >
              {createSource.isPending ? "Adding..." : "Add Source"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
