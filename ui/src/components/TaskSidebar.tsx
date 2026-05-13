import type { Task } from "../types/state";
import TaskList from "./TaskList";

interface TaskSidebarProps {
  tasks: Task[];
  selectedTaskId: string | null;
}

export default function TaskSidebar({ tasks, selectedTaskId }: TaskSidebarProps) {
  const isSingleView = selectedTaskId !== null;

  return (
    <div className="h-full flex flex-col">
      <div className="border-b border-gray-200 px-4 py-3 bg-gray-50">
        <h2 className="text-sm font-semibold text-gray-800">
          {isSingleView ? "Task Details" : "Tasks"}
        </h2>
        <p className="text-xs text-gray-500 mt-0.5">
          {isSingleView
            ? `Task ${selectedTaskId}`
            : `${tasks.length} task${tasks.length !== 1 ? 's' : ''}`}
        </p>
      </div>
      <div className="flex-1 overflow-y-auto p-4">
        <TaskList tasks={tasks} />
      </div>
    </div>
  );
}
