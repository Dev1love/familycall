import { Virtuoso } from 'react-virtuoso'
import { useAuthStore } from '../stores/authStore'

interface Message {
  id: string
  content: string
  sender_id: string
  sender: { username: string }
  created_at: string
  edited_at?: string
}

export default function MessageList({ messages }: { messages: Message[] }) {
  const user = useAuthStore((s) => s.user)

  return (
    <Virtuoso
      className="message-list"
      data={messages}
      initialTopMostItemIndex={messages.length - 1}
      followOutput="smooth"
      itemContent={(_, msg) => (
        <div className={`message ${msg.sender_id === user?.id ? 'own' : ''}`}>
          <span className="message-sender">{msg.sender.username}</span>
          <p className="message-content">{msg.content}</p>
          <span className="message-time">
            {new Date(msg.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
            {msg.edited_at && ' (edited)'}
          </span>
        </div>
      )}
    />
  )
}
