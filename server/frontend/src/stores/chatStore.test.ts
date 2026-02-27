import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useChatStore, type Chat } from './chatStore'

// Mock localStorage for authStore dependency
Object.defineProperty(globalThis, 'localStorage', {
  value: {
    getItem: vi.fn(() => 'fake-token'),
    setItem: vi.fn(),
    removeItem: vi.fn(),
    clear: vi.fn(),
  },
})

// Mock fetch for apiFetch
const mockFetch = vi.fn()
globalThis.fetch = mockFetch

const sampleChats: Chat[] = [
  {
    id: 'chat-1',
    type: 'group',
    name: 'Family',
    members: [
      { user_id: 'u1', user: { id: 'u1', username: 'Alice' } },
      { user_id: 'u2', user: { id: 'u2', username: 'Bob' } },
    ],
    unread_count: 0,
  },
  {
    id: 'chat-2',
    type: 'direct',
    members: [
      { user_id: 'u1', user: { id: 'u1', username: 'Alice' } },
      { user_id: 'u3', user: { id: 'u3', username: 'Charlie' } },
    ],
    unread_count: 2,
  },
]

describe('chatStore', () => {
  beforeEach(() => {
    useChatStore.setState({ chats: [], loading: false })
    mockFetch.mockReset()
  })

  it('starts with empty chats', () => {
    const state = useChatStore.getState()
    expect(state.chats).toEqual([])
    expect(state.loading).toBe(false)
  })

  it('fetchChats loads chats from API', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve(sampleChats),
    })

    await useChatStore.getState().fetchChats()

    const state = useChatStore.getState()
    expect(state.chats).toHaveLength(2)
    expect(state.loading).toBe(false)
    expect(state.chats[0].name).toBe('Family')
  })

  it('fetchChats handles API error', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
    })

    await useChatStore.getState().fetchChats()

    const state = useChatStore.getState()
    expect(state.chats).toEqual([])
    expect(state.loading).toBe(false)
  })

  it('addIncomingMessage updates chat', () => {
    useChatStore.setState({ chats: sampleChats })

    const newMessage = {
      id: 'msg-1',
      content: 'Hello!',
      sender_id: 'u2',
      sender: { id: 'u2', username: 'Bob' },
      created_at: new Date().toISOString(),
    }

    useChatStore.getState().addIncomingMessage('chat-1', newMessage)

    const state = useChatStore.getState()
    const chat = state.chats.find((c) => c.id === 'chat-1')!
    expect(chat.last_message).toEqual(newMessage)
    expect(chat.unread_count).toBe(1)
  })

  it('addIncomingMessage does not affect other chats', () => {
    useChatStore.setState({ chats: sampleChats })

    const newMessage = {
      id: 'msg-2',
      content: 'Hi!',
      sender_id: 'u1',
      sender: { id: 'u1', username: 'Alice' },
      created_at: new Date().toISOString(),
    }

    useChatStore.getState().addIncomingMessage('chat-1', newMessage)

    const state = useChatStore.getState()
    const chat2 = state.chats.find((c) => c.id === 'chat-2')!
    expect(chat2.unread_count).toBe(2) // unchanged
    expect(chat2.last_message).toBeUndefined()
  })
})
