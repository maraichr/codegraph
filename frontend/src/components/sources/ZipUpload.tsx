import { useRef } from "react";
import { useUpload } from "../../api/hooks";

interface Props {
  projectSlug: string;
}

export function ZipUpload({ projectSlug }: Props) {
  const fileInput = useRef<HTMLInputElement>(null);
  const upload = useUpload(projectSlug);

  const handleUpload = () => {
    const file = fileInput.current?.files?.[0];
    if (file) {
      upload.mutate(file, {
        onSuccess: () => {
          if (fileInput.current) fileInput.current.value = "";
        },
      });
    }
  };

  return (
    <div className="flex items-center gap-3">
      <input
        ref={fileInput}
        type="file"
        accept=".zip"
        className="text-sm text-gray-500 file:mr-3 file:rounded-md file:border-0 file:bg-gray-100 file:px-3 file:py-1.5 file:text-sm file:font-medium file:text-gray-700 hover:file:bg-gray-200"
      />
      <button
        type="button"
        onClick={handleUpload}
        disabled={upload.isPending}
        className="rounded-md bg-green-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-green-700 disabled:opacity-50"
      >
        {upload.isPending ? "Uploading..." : "Upload ZIP"}
      </button>
      {upload.error && <p className="text-sm text-red-600">{(upload.error as Error).message}</p>}
      {upload.isSuccess && <p className="text-sm text-green-600">Upload complete!</p>}
    </div>
  );
}
