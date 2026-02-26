import { useState, FormEvent } from 'react'

interface Props {
  onSend: (content: string) => void
  onTyping: () => void
}

export default function ChatInput({ onSend, onTyping }: Props) {
  const [text, setText] = useState('')

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault()
    const trimmed = text.trim()
    if (!trimmed) return
    onSend(trimmed)
    setText('')
  }

  return (
    <form className="chat-input" onSubmit={handleSubmit}>
      <input
        value={text}
        onChange={(e) => {
          setText(e.target.value)
          onTyping()
        }}
        placeholder="Message..."
        maxLength={4096}
        autoFocus
      />
      <button type="submit" disabled={!text.trim()}>Send</button>
    </form>
  )
}
