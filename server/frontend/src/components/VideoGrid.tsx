import { useRef, useEffect } from 'react'

interface Participant {
  userId: string
  username: string
  stream: MediaStream | null
  isLocal?: boolean
}

export default function VideoGrid({ participants }: { participants: Participant[] }) {
  const count = participants.length
  const gridClass =
    count <= 1 ? 'grid-1' :
    count <= 2 ? 'grid-2' :
    count <= 4 ? 'grid-4' : 'grid-5'

  return (
    <div className={`video-grid ${gridClass}`}>
      {participants.map((p) => (
        <VideoTile key={p.userId} participant={p} />
      ))}
    </div>
  )
}

function VideoTile({ participant }: { participant: Participant }) {
  const videoRef = useRef<HTMLVideoElement>(null)

  useEffect(() => {
    if (videoRef.current && participant.stream) {
      videoRef.current.srcObject = participant.stream
    }
  }, [participant.stream])

  return (
    <div className="video-tile">
      <video
        ref={videoRef}
        autoPlay
        playsInline
        muted={participant.isLocal}
      />
      <span className="video-label">{participant.username}</span>
    </div>
  )
}
