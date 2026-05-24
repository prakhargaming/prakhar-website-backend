package domain

type RepoDocument struct {
	Name      string
	Readme    string
	Topics    []string
	Languages map[string]int64
}
