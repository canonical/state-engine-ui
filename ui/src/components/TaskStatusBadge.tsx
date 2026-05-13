import type { Status } from "../types/state";

const STATUS_STYLES: Record<Status, string> = {
  default: "bg-zinc-700 text-zinc-400",
  hold: "bg-zinc-700 text-zinc-400",
  do: "bg-zinc-700 text-zinc-300",
  doing: "bg-blue-400/15 text-blue-400",
  done: "bg-green-400/15 text-green-400",
  abort: "bg-orange-400/15 text-orange-400",
  undo: "bg-orange-400/15 text-orange-400",
  undoing: "bg-orange-400/15 text-orange-400",
  undone: "bg-orange-400/15 text-orange-400",
  error: "bg-red-400/15 text-red-400 font-semibold",
  wait: "bg-amber-400/15 text-amber-400",
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
