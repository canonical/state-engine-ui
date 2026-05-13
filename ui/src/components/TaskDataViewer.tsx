import { useState } from "react";

interface TaskDataViewerProps {
  data: Record<string, unknown>;
}

export default function TaskDataViewer({ data }: TaskDataViewerProps) {
  const [isExpanded, setIsExpanded] = useState(true);

  if (Object.keys(data).length === 0) {
    return null;
  }

  return (
    <div className="mt-4">
      <button
        type="button"
        onClick={() => setIsExpanded((prev) => !prev)}
        className="text-xs text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300 transition-colors"
      >
        {isExpanded ? "Hide task data" : "Show task data"}
      </button>

      {isExpanded && (
        <pre className="mt-2 max-h-64 overflow-auto rounded bg-zinc-50 dark:bg-zinc-800 p-3 font-mono text-xs text-zinc-700 dark:text-zinc-300 whitespace-pre-wrap break-words">
          {JSON.stringify(data, null, 2)}
        </pre>
      )}
    </div>
  );
}
