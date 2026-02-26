package websocket

import "encoding/json"

// Signaling message types
const (
	TypeOffer        = "offer"
	TypeAnswer       = "answer"
	TypeICECandidate = "ice-candidate"
	TypeCallRequest  = "call-request"
	TypeCallAccept   = "call-accept"
	TypeCallReject   = "call-reject"
	TypeCallEnd      = "call-end"
	TypeUserOnline   = "user-online"
	TypeUserOffline  = "user-offline"
)

// Chat message types
const (
	TypeChatMessage = "chat:message"
	TypeChatSend    = "chat:send"
	TypeChatTyping  = "chat:typing"
	TypeChatRead    = "chat:mark_read"
)

// Group call message types
const (
	TypeGroupCallInvite = "call:group_invite"
	TypeGroupCallJoin   = "call:group_join"
	TypeGroupCallLeave  = "call:group_leave"
	TypeGroupCallSignal = "call:group_signal"
)

// EncodeMessage encodes a Message to JSON bytes
func EncodeMessage(msg Message) ([]byte, error) {
	return json.Marshal(msg)
}

