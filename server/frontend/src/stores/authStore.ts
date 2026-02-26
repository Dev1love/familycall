import { create } from 'zustand'

interface User {
  id: string
  username: string
}

interface AuthState {
  token: string | null
  user: User | null
  setAuth: (token: string, user: User) => void
  logout: () => void
  isAuthenticated: () => boolean
}

export const useAuthStore = create<AuthState>((set, get) => ({
  token: localStorage.getItem('authToken'),
  user: null,
  setAuth: (token, user) => {
    localStorage.setItem('authToken', token)
    set({ token, user })
  },
  logout: () => {
    localStorage.removeItem('authToken')
    set({ token: null, user: null })
  },
  isAuthenticated: () => !!get().token,
}))
