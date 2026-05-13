import { useCallback, useEffect, useState } from 'react'
import { postChangeAction } from './api'

export type SteppingState = 'paused' | 'stepping' | 'continued'

function resetStepper() {
  return {
    steppingState: 'paused' as SteppingState,
    lastSteppedTaskId: null as string | null,
    stepping: false,
    error: null as string | null,
    noRunnableTasks: false,
  }
}

export function useStepper(changeId: string | null, changeReady: boolean) {
  const [state, setState] = useState(resetStepper)

  useEffect(() => {
    setState(resetStepper())
  }, [changeId])

  const step = useCallback(async () => {
    if (!changeId || changeReady || state.stepping) return
    setState(s => ({ ...s, stepping: true, error: null, noRunnableTasks: false }))
    try {
      const allowedTask = await postChangeAction(changeId, 'step')
      if (allowedTask) {
        setState(s => ({ ...s, stepping: false, lastSteppedTaskId: allowedTask, steppingState: 'stepping' }))
      } else {
        setState(s => ({ ...s, stepping: false, noRunnableTasks: true }))
      }
    } catch (e) {
      setState(s => ({ ...s, stepping: false, error: e instanceof Error ? e.message : String(e) }))
    }
  }, [changeId, changeReady, state.stepping])

  const continue_ = useCallback(async () => {
    if (!changeId || changeReady || state.stepping) return
    setState(s => ({ ...s, stepping: true, error: null }))
    try {
      await postChangeAction(changeId, 'continue')
      setState(s => ({ ...s, stepping: false, steppingState: 'continued', lastSteppedTaskId: null }))
    } catch (e) {
      setState(s => ({ ...s, stepping: false, error: e instanceof Error ? e.message : String(e) }))
    }
  }, [changeId, changeReady, state.stepping])

  const pause = useCallback(async () => {
    if (!changeId || changeReady || state.stepping) return
    setState(s => ({ ...s, stepping: true, error: null }))
    try {
      await postChangeAction(changeId, 'pause')
      setState(s => ({ ...s, stepping: false, steppingState: 'paused', lastSteppedTaskId: null }))
    } catch (e) {
      setState(s => ({ ...s, stepping: false, error: e instanceof Error ? e.message : String(e) }))
    }
  }, [changeId, changeReady, state.stepping])

  return { ...state, step, continue: continue_, pause } as const
}
