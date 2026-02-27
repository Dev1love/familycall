import { describe, it, expect, beforeEach, vi } from 'vitest'
import { apiFetch } from './api'

const localStorageMock = {
  getItem: vi.fn(() => 'test-token'),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
}
Object.defineProperty(globalThis, 'localStorage', { value: localStorageMock })

const mockFetch = vi.fn()
globalThis.fetch = mockFetch

describe('apiFetch', () => {
  beforeEach(() => {
    mockFetch.mockReset()
    localStorageMock.getItem.mockReturnValue('test-token')
  })

  it('sends GET request with auth header', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ data: 'test' }),
    })

    const result = await apiFetch<{ data: string }>('/me')

    expect(mockFetch).toHaveBeenCalledWith('/api/me', expect.objectContaining({
      headers: expect.objectContaining({
        Authorization: 'Bearer test-token',
        'Content-Type': 'application/json',
      }),
    }))
    expect(result).toEqual({ data: 'test' })
  })

  it('sends POST request with body', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ id: '1' }),
    })

    const result = await apiFetch<{ id: string }>('/chats', {
      method: 'POST',
      body: JSON.stringify({ name: 'Test' }),
    })

    expect(result).toEqual({ id: '1' })
    expect(mockFetch).toHaveBeenCalledWith('/api/chats', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ name: 'Test' }),
    }))
  })

  it('throws on non-ok response', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 401,
    })

    await expect(apiFetch('/me')).rejects.toThrow('API error: 401')
  })

  it('throws on network error', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network error'))

    await expect(apiFetch('/me')).rejects.toThrow('Network error')
  })
})
