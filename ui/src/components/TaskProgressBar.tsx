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
        <div className="h-2 w-full rounded-full bg-zinc-200 dark:bg-zinc-700">
          <div className="h-2 w-1/3 rounded-full bg-blue-500 animate-pulse" />
        </div>
        <p className="mt-1 text-xs text-zinc-500 dark:text-zinc-400">
          In progress...
        </p>
      </div>
    );
  }

  const pct = Math.round((progress.done / progress.total) * 100);

  return (
    <div className="mt-2">
      <div className="h-2 w-full rounded-full bg-zinc-200 dark:bg-zinc-700">
        <div
          className="h-2 rounded-full bg-blue-500 transition-all duration-300"
          style={{ width: `${pct}%` }}
        />
      </div>
      <p className="mt-1 text-xs text-zinc-600 dark:text-zinc-400">
        {progress.label} {pct}% ({progress.done}/{progress.total})
      </p>
    </div>
  );
}
