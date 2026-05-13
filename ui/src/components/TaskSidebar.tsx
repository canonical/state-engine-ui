import type { Task } from "../types/state";
import TaskItem from "./TaskItem";
import TaskList from "./TaskList";

interface TaskSidebarProps {
  tasks: Task[];
  selectedTaskId: string | null;
  selectedTask: Task | null;
  onSelectTask: (id: string) => void;
  onClearSelection: () => void;
}

export default function TaskSidebar({
  tasks,
  selectedTaskId,
  selectedTask,
  onSelectTask,
  onClearSelection,
}: TaskSidebarProps) {
  const isSingleView = selectedTaskId !== null;

  return (
    <div className="h-full flex flex-col">
      <div className="border-b border-zinc-700 px-4 py-3 bg-zinc-800">
        <h2 className="text-sm font-semibold text-zinc-100">
          {isSingleView ? "Task Details" : "Tasks"}
        </h2>
        <p className="text-xs text-zinc-400 mt-0.5">
          {isSingleView
            ? `Task ${selectedTaskId}`
            : `${tasks.length} task${tasks.length !== 1 ? "s" : ""}`}
        </p>
        {isSingleView ? (
          <button
            type="button"
            onClick={onClearSelection}
            className="mt-2 text-xs text-blue-400 hover:text-blue-300 transition-colors"
          >
            &larr; Back to all tasks
          </button>
        ) : null}
      </div>
      <div className="flex-1 overflow-y-auto p-4">
        {selectedTask ? (
          <TaskItem task={selectedTask} onSelectTask={onSelectTask} />
        ) : (
          <TaskList tasks={tasks} onSelectTask={onSelectTask} />
        )}
      </div>
    </div>
  );
}
