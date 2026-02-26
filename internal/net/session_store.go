package net

// SessionStore holds all active sessions. Accessed only from the game loop
// goroutine — no mutex needed.
type SessionStore struct {
	sessions map[uint64]*Session
}

func NewSessionStore() *SessionStore {
	return &SessionStore{sessions: make(map[uint64]*Session)}
}

func (ss *SessionStore) Add(s *Session)            { ss.sessions[s.ID] = s }
func (ss *SessionStore) Remove(id uint64)           { delete(ss.sessions, id) }
func (ss *SessionStore) Get(id uint64) *Session     { return ss.sessions[id] }

// ForEach iterates all sessions. Safe to call Close() on sessions during iteration.
func (ss *SessionStore) ForEach(fn func(*Session)) {
	for _, s := range ss.sessions {
		fn(s)
	}
}

// Raw returns the underlying map for direct iteration.
// Only use from the game loop goroutine. Safe to delete during range.
func (ss *SessionStore) Raw() map[uint64]*Session {
	return ss.sessions
}
