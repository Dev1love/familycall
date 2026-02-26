import { useState, useEffect, useCallback, useRef } from 'react'
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
  const [typingUsers, setTypingUsers] = useState<string[]>([])
  const typingTimeoutRef = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map())
  const lastTypingSent = useRef<number>(0)

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
      if (msg.type === 'chat:typing' && msg.data?.chat_id === id) {
        const senderName = msg.data?.sender_name || msg.from || 'Someone'
        setTypingUsers((prev) => {
          if (!prev.includes(senderName)) return [...prev, senderName]
          return prev
        })

        // Clear after 3 seconds
        const existing = typingTimeoutRef.current.get(senderName)
        if (existing) clearTimeout(existing)
        typingTimeoutRef.current.set(
          senderName,
          setTimeout(() => {
            setTypingUsers((prev) => prev.filter((u) => u !== senderName))
            typingTimeoutRef.current.delete(senderName)
          }, 3000)
        )
      }
    },
    [id]
  )

  const { send } = useWebSocket(onWsMessage)

  // Mark as read when messages load or new ones arrive
  useEffect(() => {
    if (messages.length > 0 && id) {
      const lastMsg = messages[messages.length - 1]
      send({ type: 'chat:mark_read', data: { chat_id: id, message_id: lastMsg.id } })
    }
  }, [messages.length, id, send])

  const handleSend = async (content: string) => {
    const msg = await apiFetch<Message>(`/chats/${id}/messages`, {
      method: 'POST',
      body: JSON.stringify({ content }),
    })
    setMessages((prev) => [...prev, msg])
  }

  const handleTyping = () => {
    const now = Date.now()
    if (now - lastTypingSent.current > 2000) {
      send({ type: 'chat:typing', data: { chat_id: id } })
      lastTypingSent.current = now
    }
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
      {typingUsers.length > 0 && (
        <div className="typing-indicator">
          {typingUsers.join(', ')} typing...
        </div>
      )}
      <ChatInput onSend={handleSend} onTyping={handleTyping} />
    </div>
  )
}
