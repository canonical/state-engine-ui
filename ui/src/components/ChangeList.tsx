import TaskStatusBadge from './TaskStatusBadge'
import type { ChangeEntry } from '../types/state'

interface ChangeListProps {
  changes: ChangeEntry[]
  selectedId: string | null
  onSelect: (id: string) => void
  error: string | null
  loading: boolean
}

export default function ChangeList({ changes, selectedId, onSelect, error, loading }: ChangeListProps) {
  if (loading && changes.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-zinc-400 dark:text-zinc-500 text-sm">
        Loading changes...
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-full text-red-400 text-sm p-4 text-center">
        {error}
      </div>
    )
  }

  if (changes.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-zinc-400 dark:text-zinc-500 text-sm">
        No changes
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {changes.map((c) => (
        <button
          key={c.id}
          type="button"
          onClick={() => onSelect(c.id)}
          className={`w-full text-left rounded-lg border p-4 shadow-sm transition-colors ${
            c.id === selectedId
              ? 'border-blue-400 bg-blue-50 dark:bg-blue-400/10 dark:border-blue-400'
              : 'border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-800 hover:border-zinc-300 dark:hover:border-zinc-600'
          }`}
        >
          <div className="flex items-center gap-2">
            <TaskStatusBadge status={c.status} />
            <span className="text-xs font-mono text-zinc-500 dark:text-zinc-400">{c.kind}</span>
            <span className="text-xs text-zinc-400 dark:text-zinc-500 ml-auto">#{c.id}</span>
          </div>
          <p className="mt-1.5 text-sm font-semibold text-zinc-900 dark:text-zinc-100">{c.summary}</p>
        </button>
      ))}
    </div>
  )
}
