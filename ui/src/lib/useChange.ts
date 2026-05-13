import { useCallback, useEffect, useRef, useState } from 'react'
import { connectChangeSSE, fetchChanges } from './api'
import type { Change, ChangeEntry } from '../types/state'

export function useChangeList(pollMs = 2000) {
  const [changes, setChanges] = useState<ChangeEntry[]>([])
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

export function useChange(id: string | null) {
  const [change, setChange] = useState<Change | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const onChangeRef = useRef<(change: Change) => void>(() => {})
  const onErrorRef = useRef<(error: string) => void>(() => {})

  onChangeRef.current = (c: Change) => {
    setChange(c)
    setError(null)
    setLoading(false)
  }

  onErrorRef.current = (err: string) => {
    setError(err)
    setLoading(false)
  }

  useEffect(() => {
    if (id == null) {
      setChange(null)
      setError(null)
      setLoading(false)
      return
    }

    setLoading(true)
    const cleanup = connectChangeSSE(
      id,
      (c) => onChangeRef.current(c),
      (err) => onErrorRef.current(err),
    )

    return cleanup
  }, [id])

  return { change, error, loading }
}
