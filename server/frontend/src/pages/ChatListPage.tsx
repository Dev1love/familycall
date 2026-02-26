import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useChatStore, Chat } from '../stores/chatStore'
import { useAuthStore } from '../stores/authStore'

export default function ChatListPage() {
  const { chats, loading, fetchChats } = useChatStore()
  const user = useAuthStore((s) => s.user)
  const navigate = useNavigate()

  useEffect(() => {
    fetchChats()
  }, [fetchChats])

  const getChatName = (chat: Chat) => {
    if (chat.type === 'group') return chat.name || 'Group'
    const other = chat.members?.find((m) => m.user_id !== user?.id)
    return other?.user?.username ?? 'Chat'
  }

  if (loading) return <div className="loading">Loading...</div>

  return (
    <div className="chat-list-page">
      <header className="page-header">
        <h1>Chats</h1>
      </header>
      <ul className="chat-list">
        {chats.map((chat) => (
          <li key={chat.id} className="chat-item" onClick={() => navigate(`/chat/${chat.id}`)}>
            <div className="chat-info">
              <span className="chat-name">{getChatName(chat)}</span>
              {chat.last_message && (
                <span className="chat-preview">
                  {chat.last_message.sender.username}: {chat.last_message.content}
                </span>
              )}
            </div>
            {chat.unread_count > 0 && (
              <span className="unread-badge">{chat.unread_count}</span>
            )}
          </li>
        ))}
        {chats.length === 0 && <li className="empty">No chats yet</li>}
      </ul>
    </div>
  )
}
