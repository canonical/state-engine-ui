import type { SteppingState } from '../lib/useStepper'

interface DebuggerControlsProps {
  steppingState: SteppingState
  stepping: boolean
  noRunnableTasks: boolean
  error: string | null
  changeReady: boolean
  onStep: () => void
  onContinue: () => void
  onPause: () => void
}

export default function DebuggerControls({
  steppingState,
  stepping,
  noRunnableTasks,
  error,
  changeReady,
  onStep,
  onContinue,
  onPause,
}: DebuggerControlsProps) {
  const disabled = stepping || changeReady

  return (
    <div className="flex items-center gap-1.5">
      <button
        type="button"
        onClick={onStep}
        disabled={disabled || steppingState === 'continued'}
        className="inline-flex items-center gap-1 rounded px-2 py-1 text-xs font-medium transition-colors bg-zinc-100 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300 hover:bg-zinc-200 dark:hover:bg-zinc-700 disabled:opacity-40 disabled:cursor-not-allowed"
        title="Step: allow the next runnable task to execute"
      >
        <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M5 5l14 7-14 7V5z" />
        </svg>
        Step
      </button>
      <button
        type="button"
        onClick={onContinue}
        disabled={disabled || steppingState === 'continued'}
        className="inline-flex items-center gap-1 rounded px-2 py-1 text-xs font-medium transition-colors bg-zinc-100 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300 hover:bg-zinc-200 dark:hover:bg-zinc-700 disabled:opacity-40 disabled:cursor-not-allowed"
        title="Continue: unblock all tasks in this change"
      >
        <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M13 5l7 7-7 7M5 5l7 7-7 7" />
        </svg>
        Continue
      </button>
      <button
        type="button"
        onClick={onPause}
        disabled={disabled || steppingState === 'paused'}
        className="inline-flex items-center gap-1 rounded px-2 py-1 text-xs font-medium transition-colors bg-zinc-100 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300 hover:bg-zinc-200 dark:hover:bg-zinc-700 disabled:opacity-40 disabled:cursor-not-allowed"
        title="Pause: re-block all tasks in this change"
      >
        <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M10 9v6m4-6v6" />
        </svg>
        Pause
      </button>

      <span className={`ml-1 rounded px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${
        steppingState === 'continued'
          ? 'bg-green-100 dark:bg-green-900/40 text-green-700 dark:text-green-400'
          : steppingState === 'stepping'
            ? 'bg-blue-100 dark:bg-blue-900/40 text-blue-700 dark:text-blue-400'
            : 'bg-amber-100 dark:bg-amber-900/40 text-amber-700 dark:text-amber-400'
      }`}>
        {steppingState === 'continued' ? 'Running' : steppingState === 'stepping' ? 'Stepping' : 'Paused'}
      </span>

      {noRunnableTasks && (
        <span className="text-[10px] text-zinc-500 dark:text-zinc-400">No runnable tasks</span>
      )}

      {error && (
        <span className="text-[10px] text-red-400">{error}</span>
      )}
    </div>
  )
}
