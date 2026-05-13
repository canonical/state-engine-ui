import type { Status } from "../types/state";

const STATUS_STYLES: Record<Status, string> = {
  default: "bg-gray-100 text-gray-600",
  hold: "bg-gray-100 text-gray-600",
  do: "bg-gray-100 text-gray-700",
  doing: "bg-blue-100 text-blue-800",
  done: "bg-green-100 text-green-800",
  abort: "bg-orange-100 text-orange-800",
  undo: "bg-orange-100 text-orange-800",
  undoing: "bg-orange-100 text-orange-800",
  undone: "bg-orange-100 text-orange-800",
  error: "bg-red-100 text-red-800 font-semibold",
  wait: "bg-amber-100 text-amber-800",
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
