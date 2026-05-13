import type { Task } from "../types/state";
import TaskItem from "./TaskItem";

interface TaskListProps {
  tasks: Task[];
  onSelectTask: (id: string) => void;
}

export default function TaskList({ tasks, onSelectTask }: TaskListProps) {
  if (tasks.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-gray-400 text-sm">
        No tasks selected
      </div>
    );
  }

  return (
    <div>
      {tasks.map((task) => (
        <TaskItem key={task.id} task={task} onSelectTask={onSelectTask} />
      ))}
    </div>
  );
}
