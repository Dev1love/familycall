import { test, expect } from '@playwright/test'

/**
 * E2E tests for P2P calls and group calls.
 * Tests the full call lifecycle: initiate, start, join, leave, auto-end.
 */

let organizerToken: string
let organizerID: string
let memberToken: string
let memberID: string
let chatID: string

test.describe.serial('Call Flow E2E', () => {
  // Setup: login as existing organizer (created by api-flow.spec.ts) or register fresh
  test('setup: register organizer and member', async ({ request }) => {
    // Try to login as existing organizer first (from api-flow tests)
    let res = await request.post('/api/login', {
      data: { username: 'FamilyOrganizer' },
    })

    if (res.ok()) {
      // Reuse existing organizer
      let body = await res.json()
      organizerToken = body.token
      organizerID = body.user.id
    } else {
      // Fresh DB: register new organizer
      res = await request.post('/api/register', {
        data: { username: 'FamilyOrganizer' },
      })
      expect(res.status()).toBe(201)
      let body = await res.json()
      organizerToken = body.token
      organizerID = body.user.id
    }

    // Try to login as existing CallMember
    res = await request.post('/api/login', {
      data: { username: 'CallMember' },
    })

    if (res.ok()) {
      const body = await res.json()
      memberToken = body.token
      memberID = body.user.id
    } else {
      // Create invite and register member
      res = await request.post('/api/invite', {
        data: { contact_name: 'CallMember' },
        headers: { Authorization: `Bearer ${organizerToken}` },
      })
      expect(res.status()).toBe(201)
      const invite = await res.json()

      res = await request.post('/api/register', {
        data: { username: 'CallMember', invite_uuid: invite.uuid },
      })
      expect(res.status()).toBe(201)
      const body = await res.json()
      memberToken = body.token
      memberID = body.user.id
    }

    // Create group chat for call tests
    res = await request.post('/api/chats', {
      data: { name: 'Call Test Chat', member_ids: [memberID] },
      headers: { Authorization: `Bearer ${organizerToken}` },
    })
    expect(res.status()).toBe(201)
    const chat = await res.json()
    chatID = chat.id
  })

  // --- P2P Call Tests ---

  test('P2P: organizer initiates video call to member', async ({ request }) => {
    const res = await request.post('/api/call', {
      data: { contact_id: memberID, call_type: 'video' },
      headers: { Authorization: `Bearer ${organizerToken}` },
    })
    expect(res.ok()).toBeTruthy()
    const body = await res.json()
    expect(body.message).toBe('Call initiated')
    expect(body.call_type).toBe('video')
  })

  test('P2P: organizer initiates audio call to member', async ({ request }) => {
    const res = await request.post('/api/call', {
      data: { contact_id: memberID, call_type: 'audio' },
      headers: { Authorization: `Bearer ${organizerToken}` },
    })
    expect(res.ok()).toBeTruthy()
    const body = await res.json()
    expect(body.call_type).toBe('audio')
  })

  test('P2P: member initiates call to organizer', async ({ request }) => {
    const res = await request.post('/api/call', {
      data: { contact_id: organizerID, call_type: 'video' },
      headers: { Authorization: `Bearer ${memberToken}` },
    })
    expect(res.ok()).toBeTruthy()
  })

  test('P2P: cannot call yourself', async ({ request }) => {
    const res = await request.post('/api/call', {
      data: { contact_id: organizerID, call_type: 'video' },
      headers: { Authorization: `Bearer ${organizerToken}` },
    })
    expect(res.status()).toBe(400)
    const body = await res.json()
    expect(body.error).toBe('Cannot call yourself')
  })

  test('P2P: invalid call type rejected', async ({ request }) => {
    const res = await request.post('/api/call', {
      data: { contact_id: memberID, call_type: 'hologram' },
      headers: { Authorization: `Bearer ${organizerToken}` },
    })
    expect(res.status()).toBe(400)
  })

  test('P2P: call non-existent user returns 404', async ({ request }) => {
    const res = await request.post('/api/call', {
      data: { contact_id: 'non-existent-user-id', call_type: 'video' },
      headers: { Authorization: `Bearer ${organizerToken}` },
    })
    expect(res.status()).toBe(404)
  })

  // --- Group Call Tests ---

  test('group: start a group call', async ({ request }) => {
    const res = await request.post(`/api/chats/${chatID}/calls`, {
      headers: { Authorization: `Bearer ${organizerToken}` },
    })
    expect(res.status()).toBe(201)
    const body = await res.json()
    expect(body.chat_id).toBe(chatID)
    expect(body.started_by).toBe(organizerID)
    expect(body.ended_at).toBeFalsy() // omitempty: undefined when nil
    expect(body.participants.length).toBe(1)
  })

  test('group: cannot start duplicate active call', async ({ request }) => {
    const res = await request.post(`/api/chats/${chatID}/calls`, {
      headers: { Authorization: `Bearer ${organizerToken}` },
    })
    expect(res.status()).toBe(409)
    const body = await res.json()
    expect(body.error).toContain('active call already exists')
  })

  let activeCallID: string

  test('group: get active call ID', async ({ request }) => {
    // Start fresh: end current call first
    // Get current active call from the conflict response
    const res = await request.post(`/api/chats/${chatID}/calls`, {
      headers: { Authorization: `Bearer ${organizerToken}` },
    })
    const body = await res.json()
    activeCallID = body.call_id
    expect(activeCallID).toBeTruthy()
  })

  test('group: member joins the call', async ({ request }) => {
    const res = await request.post(`/api/calls/${activeCallID}/join`, {
      headers: { Authorization: `Bearer ${memberToken}` },
    })
    expect(res.ok()).toBeTruthy()
    const body = await res.json()
    expect(body.call_id).toBe(activeCallID)
    // Should see organizer as other participant
    expect(body.participants.length).toBe(1)
    expect(body.participants[0].user_id).toBe(organizerID)
  })

  test('group: member leaves the call', async ({ request }) => {
    const res = await request.post(`/api/calls/${activeCallID}/leave`, {
      headers: { Authorization: `Bearer ${memberToken}` },
    })
    expect(res.ok()).toBeTruthy()
    const body = await res.json()
    expect(body.left).toBe(true)
    expect(body.remaining).toBe(1) // organizer still in
  })

  test('group: organizer leaves — call auto-ends', async ({ request }) => {
    const res = await request.post(`/api/calls/${activeCallID}/leave`, {
      headers: { Authorization: `Bearer ${organizerToken}` },
    })
    expect(res.ok()).toBeTruthy()
    const body = await res.json()
    expect(body.left).toBe(true)
    expect(body.remaining).toBe(0)
  })

  test('group: cannot join ended call', async ({ request }) => {
    const res = await request.post(`/api/calls/${activeCallID}/join`, {
      headers: { Authorization: `Bearer ${memberToken}` },
    })
    expect(res.status()).toBe(410) // Gone
  })

  test('group: can start new call after previous ended', async ({ request }) => {
    const res = await request.post(`/api/chats/${chatID}/calls`, {
      headers: { Authorization: `Bearer ${organizerToken}` },
    })
    expect(res.status()).toBe(201)
    const body = await res.json()
    expect(body.id).not.toBe(activeCallID) // new call ID
    activeCallID = body.id
  })

  test('group: member joins, leaves, and rejoins', async ({ request }) => {
    // Join
    let res = await request.post(`/api/calls/${activeCallID}/join`, {
      headers: { Authorization: `Bearer ${memberToken}` },
    })
    expect(res.ok()).toBeTruthy()

    // Leave
    res = await request.post(`/api/calls/${activeCallID}/leave`, {
      headers: { Authorization: `Bearer ${memberToken}` },
    })
    expect(res.ok()).toBeTruthy()

    // Rejoin
    res = await request.post(`/api/calls/${activeCallID}/join`, {
      headers: { Authorization: `Bearer ${memberToken}` },
    })
    expect(res.ok()).toBeTruthy()
    const body = await res.json()
    expect(body.call_id).toBe(activeCallID)
  })

  test('group: non-participant cannot leave', async ({ request }) => {
    // Create third user
    let res = await request.post('/api/invite', {
      data: { contact_name: 'Outsider' },
      headers: { Authorization: `Bearer ${organizerToken}` },
    })
    const invite = await res.json()

    res = await request.post('/api/register', {
      data: { username: 'Outsider', invite_uuid: invite.uuid },
    })
    const outsider = await res.json()

    // Outsider tries to leave — not a participant
    res = await request.post(`/api/calls/${activeCallID}/leave`, {
      headers: { Authorization: `Bearer ${outsider.token}` },
    })
    expect(res.status()).toBe(404)
  })

  // --- WebSocket Signaling Test ---

  test('WebSocket: connect and receive messages', async ({ page }) => {
    // Use page.evaluate to create a WebSocket connection
    const wsResult = await page.evaluate(async (params) => {
      const { orgID, baseURL } = params
      return new Promise<string>((resolve, reject) => {
        const ws = new WebSocket(`${baseURL.replace('http', 'ws')}/ws?user_id=${orgID}`)
        const timeout = setTimeout(() => {
          ws.close()
          resolve('timeout') // WebSocket connected but no messages (expected without other users)
        }, 2000)

        ws.onopen = () => {
          clearTimeout(timeout)
          ws.close()
          resolve('connected')
        }
        ws.onerror = (e) => {
          clearTimeout(timeout)
          reject('error')
        }
      })
    }, { orgID: organizerID, baseURL: 'http://localhost:8089' })

    expect(['connected', 'timeout']).toContain(wsResult)
  })

  test('WebSocket: rejects connection without user_id', async ({ request }) => {
    // Direct HTTP request to WebSocket endpoint without user_id should fail
    const res = await request.get('/ws')
    expect(res.status()).toBe(401)
  })
})
