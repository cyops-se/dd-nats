package types

type Context struct {
	Name    string
	Cmd     string
	Wdir    string
	Trace   bool
	Version bool
	Url     string
	Port    int
	Id      string
	BindIP  string

	// SetFlags holds the names of the command line flags that were explicitly
	// provided. Used to let command line arguments take precedence over
	// values stored in the settings file.
	SetFlags map[string]bool
}

// FlagSet returns true if the named command line flag was explicitly provided
func (ctx *Context) FlagSet(name string) bool {
	return ctx != nil && ctx.SetFlags != nil && ctx.SetFlags[name]
}
