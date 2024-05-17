package model

// ESIStatus represents the current game server status.
type ESIStatus struct {
	StatusCode   int
	PlayerCount  int
	ErrorMessage string
}

func (s ESIStatus) IsOK() bool {
	return s.StatusCode < 400
}
