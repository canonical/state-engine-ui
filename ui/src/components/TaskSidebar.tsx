import type { Task } from "../types/state";
import TaskItem from "./TaskItem";
import TaskList from "./TaskList";

interface TaskSidebarProps {
  tasks: Task[];
  selectedTaskId: string | null;
  selectedTask: Task | null;
  onClearSelection: () => void;
}

export default function TaskSidebar({
  tasks,
  selectedTaskId,
  selectedTask,
  onClearSelection,
}: TaskSidebarProps) {
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
        {isSingleView ? (
          <button
            type="button"
            onClick={onClearSelection}
            className="mt-2 text-xs text-blue-600 hover:text-blue-800 transition-colors"
          >
            &larr; Back to all tasks
          </button>
        ) : null}
      </div>
      <div className="flex-1 overflow-y-auto p-4">
        {selectedTask ? <TaskItem task={selectedTask} /> : <TaskList tasks={tasks} />}
      </div>
    </div>
  );
}
