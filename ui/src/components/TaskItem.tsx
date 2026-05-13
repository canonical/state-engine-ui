import { memo } from "react";
import type { Task } from "../types/state";
import TaskStatusBadge from "./TaskStatusBadge";
import TaskProgressBar from "./TaskProgressBar";
import TaskMetadataRow from "./TaskMetadataRow";
import TaskLogViewer from "./TaskLogViewer";

interface TaskItemProps {
  task: Task;
}

const TaskItem = memo(function TaskItem({ task }: TaskItemProps) {
  const isError = task.status === "error";

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm mb-3 last:mb-0">
      <div className="flex items-center gap-3">
        <TaskStatusBadge status={task.status} />
        <span className="text-xs font-mono text-gray-500">{task.kind}</span>
      </div>

      <p className="mt-2 text-sm font-semibold text-gray-900">{task.summary}</p>

      <TaskProgressBar status={task.status} progress={task.progress} />

      <TaskMetadataRow task={task} />

      <TaskLogViewer logs={task.log} defaultExpanded={isError} />
    </div>
  );
});

export default TaskItem;
