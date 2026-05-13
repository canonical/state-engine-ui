export type Status =
  | "default"
  | "hold"
  | "do"
  | "doing"
  | "done"
  | "abort"
  | "undo"
  | "undoing"
  | "undone"
  | "error"
  | "wait";

export interface TaskProgress {
  label: string;
  done: number;
  total: number;
}

export interface Task {
  id: string;
  kind: string;
  summary: string;
  status: Status;
  waitedStatus?: "done" | "undone";
  clean: boolean;
  progress: TaskProgress | null;
  data: Record<string, unknown>;
  waitFor: string[];
  haltTasks: string[];
  lanes: number[];
  log: string[];
  change: string;
  spawnTime: string;
  readyTime: string | null;
  doingTime: number;
  undoingTime: number;
  atTime: string | null;
}

export interface ChangeEntry {
  id: string;
  kind: string;
  summary: string;
  status: Status;
  ready: boolean;
  spawnTime: string;
}

export interface Change {
  id: string;
  kind: string;
  summary: string;
  status: Status;
  ready: boolean;
  err: string | null;
  tasks: Task[];
}
