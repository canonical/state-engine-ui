import type { Status, TaskProgress } from "../types/state";

interface TaskProgressBarProps {
  status: Status;
  progress: TaskProgress | null;
}

export default function TaskProgressBar({
  status,
  progress,
}: TaskProgressBarProps) {
  if (status !== "doing") {
    return null;
  }

  if (progress === null) {
    return (
      <div className="mt-2">
        <div className="h-2 w-full rounded-full bg-gray-200">
          <div className="h-2 w-1/3 rounded-full bg-blue-500 animate-pulse" />
        </div>
        <p className="mt-1 text-xs text-gray-500">In progress...</p>
      </div>
    );
  }

  const pct = Math.round((progress.done / progress.total) * 100);

  return (
    <div className="mt-2">
      <div className="h-2 w-full rounded-full bg-gray-200">
        <div
          className="h-2 rounded-full bg-blue-500 transition-all duration-300"
          style={{ width: `${pct}%` }}
        />
      </div>
      <p className="mt-1 text-xs text-gray-600">
        {progress.label} {pct}% ({progress.done}/{progress.total})
      </p>
    </div>
  );
}
