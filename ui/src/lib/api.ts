import type { Change, ChangeEntry, Status, Task, TaskProgress } from '../types/state'

const API_BASE = import.meta.env.VITE_TASK_DEBUG_API ?? ''

const STATUS_MAP: Record<string, Status> = {
  Default: 'default',
  Hold: 'hold',
  Do: 'do',
  Doing: 'doing',
  Done: 'done',
  Abort: 'abort',
  Undo: 'undo',
  Undoing: 'undoing',
  Undone: 'undone',
  Error: 'error',
  Wait: 'wait',
}

function mapStatus(raw: string): Status {
  return STATUS_MAP[raw] ?? 'default'
}

function parseProgress(raw: { label: string; done: number; total: number }): TaskProgress | null {
  if (raw.label === '' && raw.done === 0 && raw.total === 0) return null
  return { label: raw.label, done: raw.done, total: raw.total }
}

function parseTask(raw: Record<string, unknown>): Task {
  const progress = parseProgress(raw.progress as { label: string; done: number; total: number })
  return {
    id: raw.id as string,
    kind: raw.kind as string,
    summary: raw.summary as string,
    status: mapStatus(raw.status as string),
    clean: (raw.clean as boolean) ?? false,
    progress,
    data: (raw.data as Record<string, unknown>) ?? {},
    waitFor: (raw.wait_tasks as string[]) ?? [],
    haltTasks: (raw.halt_tasks as string[]) ?? [],
    lanes: (raw.lanes as number[]) ?? [],
    log: (raw.log as string[]) ?? [],
    change: (raw.change_id as string) ?? '',
    spawnTime: (raw.spawn_time as string) ?? '',
    readyTime: (raw.ready_time as string) ?? null,
    doingTime: raw.doing_time != null ? Math.round((raw.doing_time as number) / 1e6) : 0,
    undoingTime: raw.undoing_time != null ? Math.round((raw.undoing_time as number) / 1e6) : 0,
    atTime: (raw.at_time as string) ?? null,
  }
}

async function fetchJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`)
  if (!res.ok) throw new Error(`cannot fetch ${path}: ${res.status}`)
  return res.json()
}

export async function fetchChanges(): Promise<ChangeEntry[]> {
  const raw = await fetchJSON<Record<string, unknown>[]>('/api/v1/changes')
  return raw.map(e => ({
    id: e.id as string,
    kind: e.kind as string,
    summary: e.summary as string,
    status: mapStatus(e.status as string),
    ready: e.ready as boolean,
  }))
}

export function connectChangeSSE(
  changeId: string,
  onChange: (change: Change) => void,
  onError: (error: string) => void,
): () => void {
  const es = new EventSource(`${API_BASE}/api/v1/changes/${changeId}/event`)

  function handleMessage(ev: MessageEvent) {
    if (!ev.data) return
    try {
      const data = JSON.parse(ev.data as string) as Record<string, unknown>
      const rawChanges = (data.changes as Record<string, unknown>[]) ?? []
      const rawTasks = (data.tasks as Record<string, unknown>[]) ?? []

      const rawChange = rawChanges[0]
      if (!rawChange) return

      const tasks = rawTasks.map(parseTask)
      const change: Change = {
        id: rawChange.id as string,
        kind: rawChange.kind as string,
        summary: rawChange.summary as string,
        status: mapStatus(rawChange.status as string),
        ready: rawChange.ready as boolean,
        err: (rawChange.err as string) ?? null,
        tasks,
      }
      onChange(change)
    } catch (e) {
      onError(e instanceof Error ? e.message : String(e))
    }
  }

  es.addEventListener('snapshot', handleMessage)
  es.addEventListener('task-status-changed', handleMessage)
  es.addEventListener('change-status-changed', handleMessage)
  es.addEventListener('change-task-added', handleMessage)
  es.addEventListener('change-removed', handleMessage)

  es.onerror = () => {
    onError('SSE connection error')
  }

  return () => {
    es.removeEventListener('snapshot', handleMessage)
    es.removeEventListener('task-status-changed', handleMessage)
    es.removeEventListener('change-status-changed', handleMessage)
    es.removeEventListener('change-task-added', handleMessage)
    es.removeEventListener('change-removed', handleMessage)
    es.close()
  }
}
