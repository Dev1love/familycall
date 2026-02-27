import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useAuthStore } from './authStore'

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {}
  return {
    getItem: vi.fn((key: string) => store[key] ?? null),
    setItem: vi.fn((key: string, value: string) => { store[key] = value }),
    removeItem: vi.fn((key: string) => { delete store[key] }),
    clear: vi.fn(() => { store = {} }),
  }
})()

Object.defineProperty(globalThis, 'localStorage', { value: localStorageMock })

describe('authStore', () => {
  beforeEach(() => {
    localStorageMock.clear()
    useAuthStore.setState({ token: null, user: null })
  })

  it('starts unauthenticated', () => {
    const state = useAuthStore.getState()
    expect(state.token).toBeNull()
    expect(state.user).toBeNull()
    expect(state.isAuthenticated()).toBe(false)
  })

  it('setAuth stores token and user', () => {
    const user = { id: '1', username: 'TestUser' }
    useAuthStore.getState().setAuth('test-token', user)

    const state = useAuthStore.getState()
    expect(state.token).toBe('test-token')
    expect(state.user).toEqual(user)
    expect(state.isAuthenticated()).toBe(true)
    expect(localStorageMock.setItem).toHaveBeenCalledWith('authToken', 'test-token')
  })

  it('logout clears state and localStorage', () => {
    const user = { id: '1', username: 'TestUser' }
    useAuthStore.getState().setAuth('test-token', user)
    useAuthStore.getState().logout()

    const state = useAuthStore.getState()
    expect(state.token).toBeNull()
    expect(state.user).toBeNull()
    expect(state.isAuthenticated()).toBe(false)
    expect(localStorageMock.removeItem).toHaveBeenCalledWith('authToken')
  })
})
