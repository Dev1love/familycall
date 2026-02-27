import { test, expect } from '@playwright/test'

/**
 * End-to-end API tests covering the full family onboarding and chat flow.
 * These tests run sequentially against a real backend with a fresh database.
 */

let organizerToken: string
let organizerID: string
let memberToken: string
let memberID: string
let inviteUUID: string
let chatID: string

test.describe.serial('Family Callbook E2E Flow', () => {
  test('registration is enabled on fresh database', async ({ request }) => {
    const res = await request.get('/api/registration-status')
    expect(res.ok()).toBeTruthy()
    const body = await res.json()
    expect(body.registration_enabled).toBe(true)
  })

  test('register first user (organizer)', async ({ request }) => {
    const res = await request.post('/api/register', {
      data: { username: 'FamilyOrganizer' },
    })
    expect(res.status()).toBe(201)
    const body = await res.json()
    expect(body.token).toBeTruthy()
    expect(body.user.username).toBe('FamilyOrganizer')
    organizerToken = body.token
    organizerID = body.user.id
  })

  test('registration is now disabled', async ({ request }) => {
    const res = await request.get('/api/registration-status')
    const body = await res.json()
    expect(body.registration_enabled).toBe(false)
  })

  test('cannot register without invite', async ({ request }) => {
    const res = await request.post('/api/register', {
      data: { username: 'Unauthorized' },
    })
    expect(res.status()).toBe(403)
  })

  test('organizer can get their profile', async ({ request }) => {
    const res = await request.get('/api/me', {
      headers: { Authorization: `Bearer ${organizerToken}` },
    })
    expect(res.ok()).toBeTruthy()
    const body = await res.json()
    expect(body.username).toBe('FamilyOrganizer')
    expect(body.is_first_user).toBe(true)
  })

  test('organizer creates invite', async ({ request }) => {
    const res = await request.post('/api/invite', {
      data: { contact_name: 'Grandma' },
      headers: { Authorization: `Bearer ${organizerToken}` },
    })
    expect(res.status()).toBe(201)
    const body = await res.json()
    expect(body.uuid).toBeTruthy()
    expect(body.contact_name).toBe('Grandma')
    inviteUUID = body.uuid
  })

  test('invite is publicly visible', async ({ request }) => {
    const res = await request.get(`/api/invite/${inviteUUID}`)
    expect(res.ok()).toBeTruthy()
    const body = await res.json()
    expect(body.contact_name).toBe('Grandma')
    expect(body.accepted).toBe(false)
  })

  test('register family member with invite', async ({ request }) => {
    const res = await request.post('/api/register', {
      data: { username: 'Grandma', invite_uuid: inviteUUID },
    })
    expect(res.status()).toBe(201)
    const body = await res.json()
    expect(body.user.username).toBe('Grandma')
    expect(body.user.invited_by_user_id).toBeTruthy()
    memberToken = body.token
    memberID = body.user.id
  })

  test('family member can login', async ({ request }) => {
    const res = await request.post('/api/login', {
      data: { username: 'Grandma' },
    })
    expect(res.ok()).toBeTruthy()
    const body = await res.json()
    expect(body.token).toBeTruthy()
  })

  test('family member is not first user', async ({ request }) => {
    const res = await request.get('/api/me', {
      headers: { Authorization: `Bearer ${memberToken}` },
    })
    const body = await res.json()
    expect(body.is_first_user).toBe(false)
  })

  test('organizer creates group chat', async ({ request }) => {
    const res = await request.post('/api/chats', {
      data: { name: 'Family Group', member_ids: [memberID] },
      headers: { Authorization: `Bearer ${organizerToken}` },
    })
    expect(res.status()).toBe(201)
    const body = await res.json()
    expect(body.name).toBe('Family Group')
    expect(body.members.length).toBe(2)
    chatID = body.id
  })

  test('both users see the chat', async ({ request }) => {
    // Organizer
    let res = await request.get('/api/chats', {
      headers: { Authorization: `Bearer ${organizerToken}` },
    })
    let body = await res.json()
    expect(body.length).toBe(1)
    expect(body[0].id).toBe(chatID)

    // Member
    res = await request.get('/api/chats', {
      headers: { Authorization: `Bearer ${memberToken}` },
    })
    body = await res.json()
    expect(body.length).toBe(1)
  })

  test('organizer sends messages', async ({ request }) => {
    for (const content of ['Hello family!', 'How is everyone?', 'Miss you all!']) {
      const res = await request.post(`/api/chats/${chatID}/messages`, {
        data: { content },
        headers: { Authorization: `Bearer ${organizerToken}` },
      })
      expect(res.status()).toBe(201)
    }
  })

  test('member sees all messages', async ({ request }) => {
    const res = await request.get(`/api/chats/${chatID}/messages`, {
      headers: { Authorization: `Bearer ${memberToken}` },
    })
    expect(res.ok()).toBeTruthy()
    const body = await res.json()
    expect(body.length).toBe(3)
  })

  test('member replies', async ({ request }) => {
    const res = await request.post(`/api/chats/${chatID}/messages`, {
      data: { content: 'Miss you too!' },
      headers: { Authorization: `Bearer ${memberToken}` },
    })
    expect(res.status()).toBe(201)
    const body = await res.json()
    expect(body.sender_id).toBe(memberID)
  })

  test('organizer edits a message', async ({ request }) => {
    // Get messages
    let res = await request.get(`/api/chats/${chatID}/messages`, {
      headers: { Authorization: `Bearer ${organizerToken}` },
    })
    const messages = await res.json()
    // Find organizer's message (messages are DESC)
    const orgMessage = messages.find((m: { sender_id: string }) => m.sender_id === organizerID)

    res = await request.put(`/api/messages/${orgMessage.id}`, {
      data: { content: 'Edited message' },
      headers: { Authorization: `Bearer ${organizerToken}` },
    })
    expect(res.ok()).toBeTruthy()
    const body = await res.json()
    expect(body.content).toBe('Edited message')
    expect(body.edited_at).toBeTruthy()
  })

  test('member cannot edit organizer messages', async ({ request }) => {
    let res = await request.get(`/api/chats/${chatID}/messages`, {
      headers: { Authorization: `Bearer ${memberToken}` },
    })
    const messages = await res.json()
    const orgMessage = messages.find((m: { sender_id: string }) => m.sender_id === organizerID)

    res = await request.put(`/api/messages/${orgMessage.id}`, {
      data: { content: 'Hacked!' },
      headers: { Authorization: `Bearer ${memberToken}` },
    })
    expect(res.status()).toBe(403)
  })

  test('organizer deletes a message', async ({ request }) => {
    // Send a message to delete
    let res = await request.post(`/api/chats/${chatID}/messages`, {
      data: { content: 'To be deleted' },
      headers: { Authorization: `Bearer ${organizerToken}` },
    })
    const msg = await res.json()

    res = await request.delete(`/api/messages/${msg.id}`, {
      headers: { Authorization: `Bearer ${organizerToken}` },
    })
    expect(res.ok()).toBeTruthy()
  })

  test('organizer renames the chat', async ({ request }) => {
    const res = await request.put(`/api/chats/${chatID}`, {
      data: { name: 'Our Family Chat' },
      headers: { Authorization: `Bearer ${organizerToken}` },
    })
    expect(res.ok()).toBeTruthy()
    const body = await res.json()
    expect(body.name).toBe('Our Family Chat')
  })

  test('member cannot rename the chat (not admin)', async ({ request }) => {
    const res = await request.put(`/api/chats/${chatID}`, {
      data: { name: 'Hacked Name' },
      headers: { Authorization: `Bearer ${memberToken}` },
    })
    expect(res.status()).toBe(403)
  })

  test('XSS content is escaped', async ({ request }) => {
    const res = await request.post(`/api/chats/${chatID}/messages`, {
      data: { content: '<script>alert("xss")</script>' },
      headers: { Authorization: `Bearer ${organizerToken}` },
    })
    expect(res.status()).toBe(201)
    const body = await res.json()
    expect(body.content).not.toContain('<script>')
  })

  test('TURN config is available', async ({ request }) => {
    const res = await request.get('/api/turn-config')
    expect(res.ok()).toBeTruthy()
    const body = await res.json()
    expect(body.iceServers).toBeTruthy()
  })

  test('unauthenticated requests are rejected', async ({ request }) => {
    const endpoints = ['/api/me', '/api/chats', '/api/invites/pending']
    for (const url of endpoints) {
      const res = await request.get(url)
      expect(res.status()).toBe(401)
    }
  })
})
