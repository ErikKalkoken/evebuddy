package app

// ESIStatus represents the current game server status.
type ESIStatus struct {
	PlayerCount  int
	ErrorMessage string
}

func (s ESIStatus) IsOK() bool {
	return s.ErrorMessage == ""
}
