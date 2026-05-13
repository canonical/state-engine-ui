import type { Task } from "../types/state";
import TaskItem from "./TaskItem";
import TaskList from "./TaskList";
import { useTheme } from "../context/ThemeContext";

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
  const { theme, toggleTheme } = useTheme();

  return (
    <div className="h-full flex flex-col">
      <div className="flex items-center justify-between border-b border-zinc-200 dark:border-zinc-700 px-4 py-3 bg-zinc-50 dark:bg-zinc-800">
        <div>
          <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
            {isSingleView ? "Task Details" : "Tasks"}
          </h2>
          <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-0.5">
            {isSingleView
              ? `Task ${selectedTaskId}`
              : `${tasks.length} task${tasks.length !== 1 ? "s" : ""}`}
          </p>
          {isSingleView ? (
            <button
              type="button"
              onClick={onClearSelection}
              className="mt-2 text-xs text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 transition-colors"
            >
              &larr; Back to all tasks
            </button>
          ) : null}
        </div>
        <button
          type="button"
          onClick={toggleTheme}
          className="rounded-md p-1.5 text-zinc-500 dark:text-zinc-400 hover:bg-zinc-200 dark:hover:bg-zinc-700 transition-colors"
          aria-label={`Switch to ${theme === "dark" ? "light" : "dark"} mode`}
        >
          {theme === "dark" ? (
            <svg
              className="h-4 w-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"
              />
            </svg>
          ) : (
            <svg
              className="h-4 w-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"
              />
            </svg>
          )}
        </button>
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
