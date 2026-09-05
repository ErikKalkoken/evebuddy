package app

// ESIStatus represents the current game server status.
type ESIStatus struct {
	ErrorMessage   string
	HTTPStatusCode int
	PlayerCount    int
}

func (s ESIStatus) IsOK() bool {
	return s.HTTPStatusCode >= 200 && s.HTTPStatusCode < 300
}
