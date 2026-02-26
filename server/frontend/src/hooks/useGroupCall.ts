import { useState, useRef, useCallback } from 'react'
import { apiFetch } from '../lib/api'

interface PeerState {
  userId: string
  username: string
  pc: RTCPeerConnection
  stream: MediaStream | null
}

export function useGroupCall(wsRef: React.MutableRefObject<WebSocket | null>) {
  const [peers, setPeers] = useState<Map<string, PeerState>>(new Map())
  const [localStream, setLocalStream] = useState<MediaStream | null>(null)
  const localStreamRef = useRef<MediaStream | null>(null)
  const peersRef = useRef<Map<string, PeerState>>(new Map())

  // Keep refs in sync
  const updatePeers = useCallback((updater: (prev: Map<string, PeerState>) => Map<string, PeerState>) => {
    setPeers((prev) => {
      const next = updater(prev)
      peersRef.current = next
      return next
    })
  }, [])

  const getTurnConfig = async () => {
    return apiFetch<{ ice_servers: RTCIceServer[] }>('/turn-config')
  }

  const sendSignal = useCallback((to: string, data: any) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({
        type: 'call:group_signal',
        to,
        data,
      }))
    }
  }, [wsRef])

  const createPeerConnection = useCallback(async (
    remoteUserId: string,
    remoteUsername: string,
    localStream: MediaStream,
    isInitiator: boolean,
  ) => {
    const turnConfig = await getTurnConfig()
    const pc = new RTCPeerConnection({
      iceServers: turnConfig.ice_servers,
    })

    // Add local tracks
    localStream.getTracks().forEach((track) => pc.addTrack(track, localStream))

    // Receive remote tracks
    const remoteStream = new MediaStream()
    pc.ontrack = (event) => {
      event.streams[0]?.getTracks().forEach((track) => remoteStream.addTrack(track))
      updatePeers((prev) => {
        const next = new Map(prev)
        const existing = next.get(remoteUserId)
        if (existing) {
          next.set(remoteUserId, { ...existing, stream: remoteStream })
        }
        return next
      })
    }

    // ICE candidates
    pc.onicecandidate = (event) => {
      if (event.candidate) {
        sendSignal(remoteUserId, {
          type: 'ice-candidate',
          candidate: event.candidate,
        })
      }
    }

    // Store peer
    updatePeers((prev) => {
      const next = new Map(prev)
      next.set(remoteUserId, { userId: remoteUserId, username: remoteUsername, pc, stream: remoteStream })
      return next
    })

    // If initiator, create and send offer
    if (isInitiator) {
      const offer = await pc.createOffer()
      await pc.setLocalDescription(offer)
      sendSignal(remoteUserId, {
        type: 'offer',
        sdp: pc.localDescription,
      })
    }

    return pc
  }, [sendSignal, updatePeers])

  const handleSignal = useCallback(async (fromUserId: string, data: any) => {
    const peer = peersRef.current.get(fromUserId)

    if (data.type === 'offer') {
      // Received offer — create peer connection if needed, then answer
      let pc = peer?.pc
      if (!pc && localStreamRef.current) {
        pc = await createPeerConnection(fromUserId, fromUserId, localStreamRef.current, false)
      }
      if (pc) {
        await pc.setRemoteDescription(new RTCSessionDescription(data.sdp))
        const answer = await pc.createAnswer()
        await pc.setLocalDescription(answer)
        sendSignal(fromUserId, {
          type: 'answer',
          sdp: pc.localDescription,
        })
      }
    } else if (data.type === 'answer') {
      if (peer?.pc) {
        await peer.pc.setRemoteDescription(new RTCSessionDescription(data.sdp))
      }
    } else if (data.type === 'ice-candidate') {
      if (peer?.pc) {
        await peer.pc.addIceCandidate(new RTCIceCandidate(data.candidate))
      }
    }
  }, [createPeerConnection, sendSignal])

  const startLocalStream = useCallback(async () => {
    const stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: true })
    setLocalStream(stream)
    localStreamRef.current = stream
    return stream
  }, [])

  const joinCall = useCallback(async (callId: string) => {
    const stream = await startLocalStream()
    const result = await apiFetch<{ call_id: string; participants: { user_id: string; user: { username: string } }[] }>(
      `/calls/${callId}/join`,
      { method: 'POST' }
    )

    // Create peer connection with each existing participant (we are the initiator)
    for (const p of result.participants) {
      await createPeerConnection(p.user_id, p.user?.username || p.user_id, stream, true)
    }

    return result
  }, [startLocalStream, createPeerConnection])

  const leaveCall = useCallback(async (callId: string) => {
    // Close all peer connections
    peersRef.current.forEach((peer) => peer.pc.close())
    updatePeers(() => new Map())

    // Stop local stream
    localStreamRef.current?.getTracks().forEach((track) => track.stop())
    setLocalStream(null)
    localStreamRef.current = null

    await apiFetch(`/calls/${callId}/leave`, { method: 'POST' })
  }, [updatePeers])

  const handlePeerJoin = useCallback(async (_userId: string, _username: string) => {
    // A new peer joined — they will send us an offer, we just prepare
    // The offer handling in handleSignal will create the peer connection
  }, [])

  const handlePeerLeave = useCallback((userId: string) => {
    const peer = peersRef.current.get(userId)
    if (peer) {
      peer.pc.close()
      updatePeers((prev) => {
        const next = new Map(prev)
        next.delete(userId)
        return next
      })
    }
  }, [updatePeers])

  return {
    peers,
    localStream,
    joinCall,
    leaveCall,
    handleSignal,
    handlePeerJoin,
    handlePeerLeave,
  }
}
