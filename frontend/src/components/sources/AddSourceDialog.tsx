import { useState } from "react";
import { useCreateSource } from "../../api/hooks";
import { Button } from "../ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../ui/dialog";
import { Input } from "../ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../ui/select";

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
    <Dialog open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Source</DialogTitle>
          <DialogDescription>Connect a code source to this project.</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <label className="block">
            <span className="block text-sm font-medium text-foreground">Name</span>
            <Input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="mt-1"
              required
            />
          </label>
          <div>
            <span className="block text-sm font-medium text-foreground">Type</span>
            <Select value={sourceType} onValueChange={setSourceType}>
              <SelectTrigger className="mt-1">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="git">Git Repository</SelectItem>
                <SelectItem value="upload">File Upload</SelectItem>
                <SelectItem value="database">Database</SelectItem>
                <SelectItem value="filesystem">Filesystem</SelectItem>
              </SelectContent>
            </Select>
          </div>
          {sourceType === "git" && (
            <label className="block">
              <span className="block text-sm font-medium text-foreground">Repository URL</span>
              <Input
                type="text"
                value={connectionUri}
                onChange={(e) => setConnectionUri(e.target.value)}
                placeholder="https://gitlab.com/group/repo"
                className="mt-1"
              />
            </label>
          )}
          {createSource.error && (
            <p className="text-sm text-destructive">{(createSource.error as Error).message}</p>
          )}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={createSource.isPending}>
              {createSource.isPending ? "Adding..." : "Add Source"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
