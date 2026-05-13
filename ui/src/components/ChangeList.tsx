import TaskStatusBadge from './TaskStatusBadge'

interface ChangeListProps {
  changes: { id: string; kind: string; summary: string; status: string; ready: boolean; err?: string }[]
  selectedId: string | null
  onSelect: (id: string) => void
  error: string | null
  loading: boolean
}

export default function ChangeList({ changes, selectedId, onSelect, error, loading }: ChangeListProps) {
  if (loading && changes.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-gray-400 text-sm">
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
      <div className="flex items-center justify-center h-full text-gray-400 text-sm">
        No changes
      </div>
    )
  }

  return (
    <div className="space-y-2">
      {changes.map((c) => (
        <button
          key={c.id}
          type="button"
          onClick={() => onSelect(c.id)}
          className={`w-full text-left rounded-lg border p-3 transition-colors ${
            c.id === selectedId
              ? 'border-blue-400 bg-blue-50'
              : 'border-gray-200 bg-white hover:border-gray-300'
          }`}
        >
          <div className="flex items-center gap-2">
            <TaskStatusBadge status={c.status.toLowerCase() as 'done'} />
            <span className="text-xs font-mono text-gray-500">{c.kind}</span>
            <span className="text-xs text-gray-400 ml-auto">#{c.id}</span>
          </div>
          <p className="mt-1 text-sm font-semibold text-gray-900">{c.summary}</p>
          {c.err && <p className="mt-1 text-xs text-red-600">{c.err}</p>}
        </button>
      ))}
    </div>
  )
}
