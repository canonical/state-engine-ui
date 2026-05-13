import type { Status } from "../types/state";

const STATUS_STYLES: Record<Status, string> = {
  default: "bg-zinc-100 text-zinc-600 dark:bg-zinc-700 dark:text-zinc-400",
  hold: "bg-zinc-100 text-zinc-600 dark:bg-zinc-700 dark:text-zinc-400",
  do: "bg-zinc-100 text-zinc-700 dark:bg-white/15 dark:text-white",
  doing: "bg-blue-100 text-blue-800 dark:bg-blue-400/15 dark:text-blue-400",
  done: "bg-green-100 text-green-800 dark:bg-green-400/15 dark:text-green-400",
  abort:
    "bg-orange-100 text-orange-800 dark:bg-orange-400/15 dark:text-orange-400",
  undo: "bg-orange-100 text-orange-800 dark:bg-orange-400/15 dark:text-orange-400",
  undoing:
    "bg-orange-100 text-orange-800 dark:bg-orange-400/15 dark:text-orange-400",
  undone:
    "bg-orange-100 text-orange-800 dark:bg-orange-400/15 dark:text-orange-400",
  error:
    "bg-red-100 text-red-800 dark:bg-red-400/15 dark:text-red-400 font-semibold",
  wait: "bg-amber-100 text-amber-800 dark:bg-amber-400/15 dark:text-amber-400",
};

const STATUS_LABELS: Record<Status, string> = {
  default: "Default",
  hold: "Hold",
  do: "Do",
  doing: "Doing",
  done: "Done",
  abort: "Abort",
  undo: "Undo",
  undoing: "Undoing",
  undone: "Undone",
  error: "Error",
  wait: "Wait",
};

interface TaskStatusBadgeProps {
  status: Status;
}

export default function TaskStatusBadge({ status }: TaskStatusBadgeProps) {
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_STYLES[status]}`}
      aria-label={`Status: ${STATUS_LABELS[status]}`}
    >
      {STATUS_LABELS[status]}
    </span>
  );
}
