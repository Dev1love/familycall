import { useState, useEffect, FormEvent } from 'react'
import { useAuthStore } from '../stores/authStore'

export default function LoginPage() {
  const [username, setUsername] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const { setAuth, token } = useAuthStore()

  // On mount, if we have a token, try to fetch /api/me to restore session
  useEffect(() => {
    if (token) {
      fetch('/api/me', {
        headers: { Authorization: `Bearer ${token}` },
      })
        .then((r) => {
          if (!r.ok) throw new Error('invalid token')
          return r.json()
        })
        .then((data) => {
          setAuth(token, data.user || data)
        })
        .catch(() => {
          localStorage.removeItem('authToken')
          useAuthStore.setState({ token: null })
        })
    }
  }, [])

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')

    try {
      // Try login first
      const loginRes = await fetch('/api/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username }),
      })

      if (loginRes.ok) {
        const data = await loginRes.json()
        setAuth(data.token, data.user || { id: data.user_id, username })
        return
      }

      // If login fails, try register (first user)
      const registerRes = await fetch('/api/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username }),
      })

      if (registerRes.ok) {
        const data = await registerRes.json()
        setAuth(data.token, data.user || { id: data.user_id, username })
        return
      }

      setError('Login failed. Check username or use an invite link.')
    } catch {
      setError('Connection error. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="login-page">
      <div className="login-card">
        <h1>Family Messenger</h1>
        <p className="subtitle">Stay connected with your family</p>
        <form onSubmit={handleSubmit}>
          <input
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="Username"
            minLength={3}
            maxLength={100}
            required
            autoFocus
          />
          <button type="submit" disabled={loading}>
            {loading ? 'Signing in...' : 'Sign In'}
          </button>
          {error && <p className="error">{error}</p>}
        </form>
      </div>
    </div>
  )
}
