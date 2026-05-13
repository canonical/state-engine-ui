import type { Change, Status, Task } from "../types/state";

type Theme = "light" | "dark";

const SNAKE_COL_SIZE = 7;

function clusterLabel(lanes: number[]): string {
  return `[${lanes.join(" ")}]`;
}

function lanesLess(a: number[], b: number[]): number {
  const n = Math.min(a.length, b.length);
  for (let i = 0; i < n; i++) {
    if (a[i] < b[i]) return -1;
    if (a[i] > b[i]) return 1;
  }
  return a.length - b.length;
}

function sortTasks(tasks: Task[], labels: Map<Task, string>): void {
  tasks.sort((a, b) => {
    const la = labels.get(a)!;
    const lb = labels.get(b)!;
    if (la < lb) return -1;
    if (la > lb) return 1;
    return 0;
  });
}

function nodeAttrs(status: Status, theme: Theme): string[] {
  if (theme === "light") {
    switch (status) {
      case "done":
        return ["style=filled", "fillcolor=lightgreen", 'color="#22c55e"', "penwidth=1.5"];
      case "doing":
        return ["style=filled", "fillcolor=lightblue", 'color="#3b82f6"', "penwidth=1.5"];
      case "error":
        return ["style=filled", "fillcolor=mistyrose", 'color="#ef4444"', "penwidth=1.5"];
      case "undone":
        return ["style=filled", "fillcolor=moccasin", 'color="#f97316"', "penwidth=1.5"];
      case "wait":
        return ["style=filled", "fillcolor=lightyellow", 'color="#f59e0b"', "penwidth=1.5"];
      case "hold":
        return ["style=filled", "fillcolor=lightgray", 'color="#a1a1aa"', "penwidth=1.5"];
      default:
        return ["style=filled", "fillcolor=white", 'color="#a1a1aa"', "penwidth=1.5"];
    }
  }
  switch (status) {
    case "done":
      return ["style=filled", 'fillcolor="#4ade80"', 'fontcolor=black', 'color="#86efac"', "penwidth=1.5"];
    case "doing":
      return ["style=filled", 'fillcolor="#60a5fa"', 'fontcolor=black', 'color="#93c5fd"', "penwidth=1.5"];
    case "error":
      return ["style=filled", 'fillcolor="#f87171"', 'fontcolor=black', 'color="#fca5a5"', "penwidth=1.5"];
    case "undone":
      return ["style=filled", 'fillcolor="#fb923c"', 'fontcolor=black', 'color="#fdba74"', "penwidth=1.5"];
    case "wait":
      return ["style=filled", 'fillcolor="#fbbf24"', 'fontcolor=black', 'color="#fde68a"', "penwidth=1.5"];
    case "hold":
      return ["style=filled", 'fillcolor="#a1a1aa"', 'color="#52525b"', "penwidth=1.5"];
    case "do":
      return ["style=filled", 'fillcolor="#ffffff"', 'fontcolor=black', 'color="#52525b"', "penwidth=1.5"];
    default:
      return ["style=filled", 'fillcolor="#52525b"', 'color="#3f3f46"', "penwidth=1.5"];
  }
}

function clusterAttrs(theme: Theme): string {
  if (theme === "light") {
    return 'style=filled; fillcolor="#f4f4f5"; fontcolor=black; color=black';
  }
  return 'style=filled; fillcolor="#27272a"; fontcolor=white; color=white';
}

function edgeColor(theme: Theme): string {
  return theme === "light" ? "color=black" : "color=white";
}

function transitiveReduce(adj: Map<string, string[]>): Map<string, string[]> {
  const result = new Map<string, string[]>();
  for (const [u, neighbors] of adj) {
    const reachable = new Set<string>();
    const queue = [...neighbors];
    while (queue.length > 0) {
      const w = queue.pop()!;
      const next = adj.get(w);
      if (next) {
        for (const x of next) {
          if (!reachable.has(x)) {
            reachable.add(x);
            queue.push(x);
          }
        }
      }
    }
    result.set(
      u,
      neighbors.filter((v) => !reachable.has(v))
    );
  }
  return result;
}

function topoSortCluster(tasks: Task[], labels: Map<Task, string>): Task[] {
  const taskIds = new Set(tasks.map(t => t.id));
  const taskMap = new Map(tasks.map(t => [t.id, t]));
  const inDegree = new Map<string, number>();
  const adj = new Map<string, string[]>();

  for (const t of tasks) {
    inDegree.set(t.id, 0);
    adj.set(t.id, []);
  }

  for (const t of tasks) {
    for (const waitId of t.waitFor) {
      if (taskIds.has(waitId)) {
        adj.get(waitId)!.push(t.id);
        inDegree.set(t.id, (inDegree.get(t.id) || 0) + 1);
      }
    }
  }

  const queue: Task[] = [];
  for (const t of tasks) {
    if (inDegree.get(t.id) === 0) {
      queue.push(t);
    }
  }
  sortTasks(queue, labels);

  const result: Task[] = [];
  while (queue.length > 0) {
    const t = queue.shift()!;
    result.push(t);
    const nextTasks: Task[] = [];
    for (const nextId of adj.get(t.id) || []) {
      const newDeg = (inDegree.get(nextId) || 1) - 1;
      inDegree.set(nextId, newDeg);
      if (newDeg === 0) {
        nextTasks.push(taskMap.get(nextId)!);
      }
    }
    sortTasks(nextTasks, labels);
    queue.push(...nextTasks);
  }

  const resultSet = new Set(result.map(t => t.id));
  for (const t of tasks) {
    if (!resultSet.has(t.id)) result.push(t);
  }

  return result;
}

export function generateDot(change: Change, theme: Theme = "dark"): string {
  const tasks = [...change.tasks];
  const labels = new Map<Task, string>();

  for (const t of tasks) {
    labels.set(t, `${t.kind}:${t.id}`);
  }

  sortTasks(tasks, labels);

  const clusters: number[][] = [];
  const clusterTasks = new Map<string, Task[]>();
  const taskToCluster = new Map<Task, string>();

  for (const t of tasks) {
    const lanes = [...t.lanes].sort((a, b) => a - b);
    const clulabel = clusterLabel(lanes);
    if (!clusterTasks.has(clulabel)) {
      clusters.push(lanes);
    }
    clusterTasks.set(clulabel, [...(clusterTasks.get(clulabel) || []), t]);
    taskToCluster.set(t, clulabel);
  }

  clusters.sort(lanesLess);

  const clusterColumns = new Map<string, Task[][]>();
  const taskColumnIdx = new Map<string, number>();

  for (const clu of clusters) {
    const clulabel = clusterLabel(clu);
    const cluTasks = clusterTasks.get(clulabel)!;
    const sorted = topoSortCluster(cluTasks, labels);

    const columns: Task[][] = [];
    for (let i = 0; i < sorted.length; i += SNAKE_COL_SIZE) {
      columns.push(sorted.slice(i, i + SNAKE_COL_SIZE));
    }
    clusterColumns.set(clulabel, columns);

    for (let colIdx = 0; colIdx < columns.length; colIdx++) {
      for (const t of columns[colIdx]) {
        taskColumnIdx.set(t.id, colIdx);
      }
    }
  }

  const haltMap = transitiveReduce(
    (() => {
      const m = new Map<string, string[]>();
      for (const t of change.tasks) {
        for (const waitId of t.waitFor) {
          if (!m.has(waitId)) {
            m.set(waitId, []);
          }
          m.get(waitId)!.push(t.id);
        }
      }
      return m;
    })()
  );

  const lines: string[] = [];
  lines.push("digraph {");

  for (const clu of clusters) {
    const clulabel = clusterLabel(clu);
    const columns = clusterColumns.get(clulabel)!;

    lines.push(`subgraph "cluster${clulabel}" {`);
    lines.push(`${clusterAttrs(theme)}; tooltip="Lanes: ${clulabel}"`);

    for (const t of clusterTasks.get(clulabel)!) {
      const attrs = nodeAttrs(t.status, theme);
      const allAttrs = [`id="task-${t.id}"`, ...attrs];
      const attrStr = ` [${allAttrs.join(", ")}]`;
      lines.push(`  "${labels.get(t)}"${attrStr}`);
    }

    for (let colIdx = 0; colIdx < columns.length; colIdx++) {
      const col = columns[colIdx];
      const isOdd = colIdx % 2 === 1;

      const nodeNames = col.map(t => `"${labels.get(t)}"`).join("; ");
      lines.push(`  {rank=same ${nodeNames}}`);

      const orderedCol = isOdd ? [...col].reverse() : [...col];
      for (let i = 0; i < orderedCol.length - 1; i++) {
        lines.push(
          `  "${labels.get(orderedCol[i])}" -> "${labels.get(orderedCol[i + 1])}" [style=invis, weight=100]`
        );
      }
    }

    lines.push("}");
  }

  for (const t of tasks) {
    const clu = taskToCluster.get(t)!;
    const haltIds = haltMap.get(t.id) || [];
    const haltTasks = haltIds
      .map((id) => change.tasks.find((t2) => t2.id === id))
      .filter((t2): t2 is Task => t2 !== undefined);

    sortTasks(haltTasks, labels);

    for (const t2 of haltTasks) {
      const sameCluster = taskToCluster.get(t2)! === clu;
      const sameColumn =
        sameCluster && taskColumnIdx.get(t.id) === taskColumnIdx.get(t2.id);

      let attrs = edgeColor(theme);
      if (!sameCluster) {
        attrs = "style=bold, " + attrs;
      }
      if (sameColumn) {
        attrs += ", constraint=false";
      }
      const attrStr = ` [${attrs}]`;
      lines.push(`"${labels.get(t)}" -> "${labels.get(t2)}"${attrStr}`);
    }
  }

  lines.push("}");
  return lines.join("\n") + "\n";
}
