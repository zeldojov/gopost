package session

const flashPrefix = "flash:"

func (s *Session) GetValue(key string) (string, bool) {
	value, ok := s.userData[key]
	return value, ok
}

func (s *Session) SetValue(key, value string) {
	s.userData[key] = value
}

func (s *Session) DeleteValue(key string) {
	delete(s.userData, key)
}

func (s *Session) HasValue(key string) bool {
	_, ok := s.userData[key]
	return ok
}

func (sess *Session) SetFlash(key, message string) {
	sess.SetValue(flashPrefix+key, message)
}

func (sess *Session) GetFlash(key string) (string, bool) {
	key = flashPrefix + key

	message, ok := sess.GetValue(key)
	if ok {
		sess.DeleteValue(key)
	}

	return message, ok
}
