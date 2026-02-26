import { useEffect, useState, useRef } from 'react'
import { useParams, useNavigate, useSearchParams } from 'react-router-dom'
import { useAuthStore } from '../stores/authStore'
import { useGroupCall } from '../hooks/useGroupCall'
import { apiFetch } from '../lib/api'
import VideoGrid from '../components/VideoGrid'

export default function CallPage() {
  const { id: chatId } = useParams<{ id: string }>()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)
  const [callId, setCallId] = useState<string>(searchParams.get('callId') || '')
  const [joined, setJoined] = useState(false)
  const [starting, setStarting] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)

  const {
    peers,
    localStream,
    joinCall,
    leaveCall,
    handleSignal,
    handlePeerJoin,
    handlePeerLeave,
  } = useGroupCall(wsRef)

  // WebSocket connection for signaling
  useEffect(() => {
    if (!user) return

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${protocol}//${window.location.host}/ws?user_id=${user.id}`)

    ws.onmessage = (event) => {
      const msg = JSON.parse(event.data)

      switch (msg.type) {
        case 'call:group_signal':
          handleSignal(msg.from, msg.data)
          break
        case 'call:group_join':
          handlePeerJoin(msg.data.user_id, msg.data.user_id)
          break
        case 'call:group_leave':
          handlePeerLeave(msg.data.user_id)
          break
      }
    }

    wsRef.current = ws

    return () => {
      ws.close()
    }
  }, [user, handleSignal, handlePeerJoin, handlePeerLeave])

  const handleStartCall = async () => {
    setStarting(true)
    try {
      const result = await apiFetch<{ id: string }>(`/chats/${chatId}/calls`, { method: 'POST' })
      setCallId(result.id)
      await joinCall(result.id)
      setJoined(true)
    } catch (err: any) {
      // If call already active, try to get its ID
      if (err.message?.includes('409')) {
        // TODO: get active call ID
      }
    } finally {
      setStarting(false)
    }
  }

  const handleJoinCall = async () => {
    if (!callId) return
    await joinCall(callId)
    setJoined(true)
  }

  const handleLeave = async () => {
    if (callId) await leaveCall(callId)
    navigate(`/chat/${chatId}`)
  }

  const participants = [
    ...(localStream ? [{ userId: user!.id, username: 'You', stream: localStream, isLocal: true }] : []),
    ...Array.from(peers.values()).map((p) => ({
      userId: p.userId,
      username: p.username,
      stream: p.stream,
      isLocal: false,
    })),
  ]

  return (
    <div className="call-page">
      {!joined ? (
        <div className="call-join">
          <h2>Group Call</h2>
          {callId ? (
            <button onClick={handleJoinCall}>Join Call</button>
          ) : (
            <button onClick={handleStartCall} disabled={starting}>
              {starting ? 'Starting...' : 'Start Call'}
            </button>
          )}
          <button onClick={() => navigate(`/chat/${chatId}`)}>Cancel</button>
        </div>
      ) : (
        <>
          <VideoGrid participants={participants} />
          <div className="call-controls">
            <button className="end-call" onClick={handleLeave}>End Call</button>
          </div>
        </>
      )}
    </div>
  )
}
