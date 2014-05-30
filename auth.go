package admin

import (
	"net/http"
	"time"
)

func (a *Admin) isLoggedIn(req *http.Request) bool {
	cookie, err := req.Cookie("admin")
	if err != nil {
		return false
	}

	// TODO: Expire sessions
	_, ok := a.sessions[cookie.Value]
	return ok
}

func (a *Admin) logIn(rw http.ResponseWriter, username, password string) bool {
	if username != a.Username || password != a.Password {
		return false
	}
	sessKey := randString(32)
	a.sessions[sessKey] = &session{
		time:     time.Now(),
		messages: []*flashMessage{},
	}

	http.SetCookie(rw, &http.Cookie{
		Name:  "admin",
		Value: sessKey,
		Path:  a.Path,
	})
	return true
}

type session struct {
	time     time.Time
	messages []*flashMessage
}

func (s *session) addMessage(class, text string) {
	s.messages = append(s.messages, &flashMessage{class, text})
}

func (s *session) getMessages() []*flashMessage {
	messages := s.messages
	s.messages = []*flashMessage{}
	return messages
}

type flashMessage struct {
	class string
	text  string
}
