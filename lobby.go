package main

type Lobby struct {
	hosting bool
	name    string
	Ready   bool
}

func CreateLobby(name string) Lobby {
	return Lobby{hosting: true, name: name, Ready: false}
}

func (l *Lobby) Name() string {
	return l.name
}
