import { formatDuration } from "../lib/formatDuration";
import type { Task } from "../types/state";

interface TaskMetadataRowProps {
  task: Task;
}

export default function TaskMetadataRow({ task }: TaskMetadataRowProps) {
  const spawnTime = new Date(task.spawnTime).toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });

  const readyTime = task.readyTime
    ? new Date(task.readyTime).toLocaleTimeString([], {
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
      })
    : "—";

  return (
    <div className="mt-2 flex flex-wrap items-center gap-2">
      <span
        className="inline-flex items-center gap-1 rounded-full bg-zinc-100 dark:bg-zinc-700 px-2 py-1 text-xs text-zinc-600 dark:text-zinc-400"
        title={`Spawn: ${task.spawnTime}`}
      >
        🕐 {spawnTime}
      </span>

      <span
        className="inline-flex items-center gap-1 rounded-full bg-zinc-100 dark:bg-zinc-700 px-2 py-1 text-xs text-zinc-600 dark:text-zinc-400"
        title={task.readyTime ? `Ready: ${task.readyTime}` : "Not ready"}
      >
        ✅ {readyTime}
      </span>

      {task.doingTime > 0 && (
        <span
          className="inline-flex items-center gap-1 rounded-full bg-zinc-100 dark:bg-zinc-700 px-2 py-1 text-xs text-zinc-600 dark:text-zinc-400"
          title="Doing duration"
        >
          ⏱ {formatDuration(task.doingTime)}
        </span>
      )}

      {task.undoingTime > 0 && (
        <span
          className="inline-flex items-center gap-1 rounded-full bg-zinc-100 dark:bg-zinc-700 px-2 py-1 text-xs text-zinc-600 dark:text-zinc-400"
          title="Undoing duration"
        >
          ↩ {formatDuration(task.undoingTime)}
        </span>
      )}

      {(task.lanes.length > 1 ||
        (task.lanes.length === 1 && task.lanes[0] !== 0)) && (
        <span
          className="inline-flex items-center gap-1 rounded-full bg-zinc-100 dark:bg-zinc-700 px-2 py-1 text-xs text-zinc-600 dark:text-zinc-400"
          title="Execution lane"
        >
          🛤 Lane {task.lanes.join(", ")}
        </span>
      )}
    </div>
  );
}
