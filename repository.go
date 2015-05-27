package rbuild

type Repository struct {
	Name    string
	AbsPath string // Must be absolute path
	Env     []EnvItem
}
