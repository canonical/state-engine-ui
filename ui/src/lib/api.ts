import type { Change, Status, Task, TaskProgress } from '../types/state'

const API_BASE = import.meta.env.VITE_TASK_DEBUG_API ?? ''

interface RawProgress {
  label: string
  done: number
  total: number
}

interface RawTask {
  id: string
  kind: string
  summary: string
  status: string
  change_id?: string
  progress: RawProgress
  data?: Record<string, unknown>
  wait_tasks?: string[]
  halt_tasks?: string[]
  lanes?: number[]
  log?: string[]
  spawn_time?: string
  ready_time?: string
  at_time?: string
  doing_time?: number
  undoing_time?: number
  clean?: boolean
}

interface RawChange {
  id: string
  kind: string
  summary: string
  status: string
  ready: boolean
  err?: string
  spawn_time?: string
  ready_time?: string
  task_ids?: string[]
}

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

function mapProgress(raw: RawProgress): TaskProgress | null {
  if (raw.label === '' && raw.done === 0 && raw.total === 0) return null
  return { label: raw.label, done: raw.done, total: raw.total }
}

function mapTask(raw: RawTask): Task {
  return {
    id: raw.id,
    kind: raw.kind,
    summary: raw.summary,
    status: mapStatus(raw.status),
    clean: raw.clean ?? false,
    progress: mapProgress(raw.progress),
    data: raw.data ?? {},
    waitFor: raw.wait_tasks ?? [],
    haltTasks: raw.halt_tasks ?? [],
    lanes: raw.lanes ?? [],
    log: raw.log ?? [],
    change: raw.change_id ?? '',
    spawnTime: raw.spawn_time ?? '',
    readyTime: raw.ready_time ?? null,
    doingTime: raw.doing_time != null ? Math.round(raw.doing_time / 1e6) : 0,
    undoingTime: raw.undoing_time != null ? Math.round(raw.undoing_time / 1e6) : 0,
    atTime: raw.at_time ?? null,
  }
}

async function fetchJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`)
  if (!res.ok) throw new Error(`cannot fetch ${path}: ${res.status}`)
  return res.json()
}

export async function fetchChanges(): Promise<RawChange[]> {
  return fetchJSON<RawChange[]>('/changes')
}

export async function fetchChange(id: string): Promise<RawChange> {
  return fetchJSON<RawChange>(`/changes/${id}`)
}

export async function fetchChangeTasks(id: string): Promise<RawTask[]> {
  return fetchJSON<RawTask[]>(`/changes/${id}/tasks`)
}

export async function fetchChangeWithTasks(id: string): Promise<Change> {
  const [rawChange, rawTasks] = await Promise.all([
    fetchChange(id),
    fetchChangeTasks(id),
  ])
  return {
    id: rawChange.id,
    kind: rawChange.kind,
    summary: rawChange.summary,
    tasks: rawTasks.map(mapTask),
  }
}
