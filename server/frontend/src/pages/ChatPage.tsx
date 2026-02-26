import { useState, useEffect, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { apiFetch } from '../lib/api'
import { useAuthStore } from '../stores/authStore'
import { useWebSocket } from '../hooks/useWebSocket'
import MessageList from '../components/MessageList'
import ChatInput from '../components/ChatInput'

interface Message {
  id: string
  content: string
  sender_id: string
  sender: { id: string; username: string }
  created_at: string
  edited_at?: string
}

interface ChatInfo {
  id: string
  type: 'direct' | 'group'
  name?: string
  members: { user_id: string; user: { id: string; username: string } }[]
}

export default function ChatPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)
  const [messages, setMessages] = useState<Message[]>([])
  const [chat, setChat] = useState<ChatInfo | null>(null)

  useEffect(() => {
    if (!id) return
    apiFetch<ChatInfo>(`/chats/${id}`).then(setChat)
    apiFetch<Message[]>(`/chats/${id}/messages`).then((msgs) =>
      setMessages(msgs.reverse())
    )
  }, [id])

  const onWsMessage = useCallback(
    (msg: any) => {
      if (msg.type === 'chat:message' && msg.data?.chat_id === id) {
        setMessages((prev) => [...prev, msg.data.message])
      }
    },
    [id]
  )

  const { send } = useWebSocket(onWsMessage)

  const handleSend = async (content: string) => {
    const msg = await apiFetch<Message>(`/chats/${id}/messages`, {
      method: 'POST',
      body: JSON.stringify({ content }),
    })
    setMessages((prev) => [...prev, msg])
  }

  const handleTyping = () => {
    send({ type: 'chat:typing', data: { chat_id: id } })
  }

  const getChatName = () => {
    if (!chat) return 'Chat'
    if (chat.type === 'group') return chat.name || 'Group'
    const other = chat.members?.find((m) => m.user_id !== user?.id)
    return other?.user?.username ?? 'Chat'
  }

  return (
    <div className="chat-page">
      <header className="chat-header">
        <button className="back-btn" onClick={() => navigate('/')}>&#8592;</button>
        <h2>{getChatName()}</h2>
        {chat?.type === 'group' && (
          <button className="call-btn" onClick={() => navigate(`/chat/${id}/call`)}>Call</button>
        )}
      </header>
      <MessageList messages={messages} />
      <ChatInput onSend={handleSend} onTyping={handleTyping} />
    </div>
  )
}
