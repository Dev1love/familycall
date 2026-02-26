import { create } from 'zustand'
import { apiFetch } from '../lib/api'

interface ChatMember {
  user_id: string
  user: { id: string; username: string }
}

interface Message {
  id: string
  content: string
  sender_id: string
  sender: { id: string; username: string }
  created_at: string
  edited_at?: string
}

export interface Chat {
  id: string
  type: 'direct' | 'group'
  name?: string
  members: ChatMember[]
  last_message?: Message
  unread_count: number
}

interface ChatState {
  chats: Chat[]
  loading: boolean
  fetchChats: () => Promise<void>
  addIncomingMessage: (chatId: string, message: Message) => void
}

export const useChatStore = create<ChatState>((set) => ({
  chats: [],
  loading: false,
  fetchChats: async () => {
    set({ loading: true })
    try {
      const chats = await apiFetch<Chat[]>('/chats')
      set({ chats, loading: false })
    } catch {
      set({ loading: false })
    }
  },
  addIncomingMessage: (chatId, message) => {
    set((state) => ({
      chats: state.chats.map((c) =>
        c.id === chatId
          ? { ...c, last_message: message, unread_count: c.unread_count + 1 }
          : c
      ),
    }))
  },
}))
