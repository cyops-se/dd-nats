package types

type Context struct {
	Name    string
	Cmd     string
	Wdir    string
	Trace   bool
	Version bool
	NatsUrl string
	Port    int
	Id      string
}
