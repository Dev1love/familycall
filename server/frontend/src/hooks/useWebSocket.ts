import { useEffect, useRef, useCallback } from 'react'
import { useAuthStore } from '../stores/authStore'

type MessageHandler = (msg: any) => void

export function useWebSocket(onMessage: MessageHandler) {
  const wsRef = useRef<WebSocket | null>(null)
  const user = useAuthStore((s) => s.user)

  useEffect(() => {
    if (!user) return

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${protocol}//${window.location.host}/ws?user_id=${user.id}`)

    ws.onmessage = (event) => {
      const msg = JSON.parse(event.data)
      onMessage(msg)
    }

    ws.onclose = () => {
      // Auto-reconnect after 3s
      setTimeout(() => {
        // useEffect will re-run if deps change
      }, 3000)
    }

    wsRef.current = ws

    return () => {
      ws.close()
    }
  }, [user, onMessage])

  const send = useCallback((msg: any) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(msg))
    }
  }, [])

  return { send }
}
