import type { Change, Status, Task } from "../types/state";

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

function nodeAttrs(status: Status): string[] {
  switch (status) {
    case "done":
      return ["style=filled", 'fillcolor="#4ade80"']; // green-400
    case "doing":
      return ["style=filled", 'fillcolor="#60a5fa"']; // blue-400
    case "error":
      return ["style=filled", 'fillcolor="#f87171"']; // red-400
    case "undone":
      return ["style=filled", 'fillcolor="#fb923c"']; // orange-400
    case "wait":
      return ["style=filled", 'fillcolor="#fbbf24"']; // amber-400
    case "hold":
      return ["style=filled", 'fillcolor="#a1a1aa"']; // zinc-400
    default:
      return ["style=filled", 'fillcolor="#52525b"']; // zinc-600
  }
}

export function generateDot(change: Change): string {
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

  const haltMap = new Map<string, string[]>();
  for (const t of change.tasks) {
    for (const waitId of t.waitFor) {
      if (!haltMap.has(waitId)) {
        haltMap.set(waitId, []);
      }
      haltMap.get(waitId)!.push(t.id);
    }
  }

  const lines: string[] = [];
  lines.push("digraph {");

  for (const clu of clusters) {
    const clulabel = clusterLabel(clu);
    lines.push(`subgraph "cluster${clulabel}" {`);
    lines.push(
      `style=filled; fillcolor="#27272a"; fontcolor=white; color=white; tooltip="Lanes: ${clulabel}"`,
    );
    for (const t of clusterTasks.get(clulabel)!) {
      const attrs = nodeAttrs(t.status);
      const allAttrs = [`id="task-${t.id}"`, ...attrs];
      const attrStr = ` [${allAttrs.join(", ")}]`;
      lines.push(`  "${labels.get(t)}"${attrStr}`);
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
      let attrs = "color=white";
      if (taskToCluster.get(t2)! !== clu) {
        attrs = "style=bold, " + attrs;
      }
      const attrStr = ` [${attrs}]`;
      lines.push(`"${labels.get(t)}" -> "${labels.get(t2)}"${attrStr}`);
    }
  }

  lines.push("}");
  return lines.join("\n") + "\n";
}
