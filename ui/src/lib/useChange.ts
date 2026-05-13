import { useCallback, useEffect, useRef, useState } from 'react'
import { fetchChangeWithTasks, fetchChanges } from './api'
import type { Change } from '../types/state'

interface RawChangeSummary {
  id: string
  kind: string
  summary: string
  status: string
  ready: boolean
  err?: string
  task_ids?: string[]
}

export function useChangeList(pollMs = 2000) {
  const [changes, setChanges] = useState<RawChangeSummary[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const load = useCallback(async () => {
    try {
      const data = await fetchChanges()
      setChanges(data)
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
    function schedule() {
      timerRef.current = setTimeout(() => {
        load().finally(schedule)
      }, pollMs)
    }
    schedule()
    return () => {
      if (timerRef.current != null) clearTimeout(timerRef.current)
    }
  }, [load, pollMs])

  return { changes, error, loading, reload: load }
}

export function useChange(id: string | null, pollMs = 1000) {
  const [change, setChange] = useState<Change | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const load = useCallback(async () => {
    if (id == null) return
    setLoading(true)
    try {
      const data = await fetchChangeWithTasks(id)
      setChange(data)
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => {
    if (id == null) {
      setChange(null)
      setError(null)
      setLoading(false)
      return
    }
    load()
    function schedule() {
      timerRef.current = setTimeout(() => {
        load().finally(schedule)
      }, pollMs)
    }
    schedule()
    return () => {
      if (timerRef.current != null) clearTimeout(timerRef.current)
    }
  }, [id, load, pollMs])

  return { change, error, loading, reload: load }
}
