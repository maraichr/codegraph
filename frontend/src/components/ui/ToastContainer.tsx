import { type ToastVariant, useToastStore } from "../../stores/toast";

const variantClasses: Record<ToastVariant, string> = {
  error: "bg-red-600",
  success: "bg-green-600",
  info: "bg-blue-600",
};

export function ToastContainer() {
  const toasts = useToastStore((s) => s.toasts);
  const removeToast = useToastStore((s) => s.removeToast);

  if (toasts.length === 0) return null;

  return (
    <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2">
      {toasts.map((toast) => (
        <div
          key={toast.id}
          className={`${variantClasses[toast.variant]} flex items-start gap-2 rounded-lg px-4 py-3 text-sm text-white shadow-lg`}
        >
          <span className="flex-1">{toast.message}</span>
          <button
            type="button"
            onClick={() => removeToast(toast.id)}
            className="ml-2 shrink-0 opacity-70 hover:opacity-100"
          >
            &times;
          </button>
        </div>
      ))}
    </div>
  );
}
