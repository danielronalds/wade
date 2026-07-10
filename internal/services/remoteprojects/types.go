package remoteprojects

// TODO: Review properly

type Project struct {
	Name          string `json:"name"`
	NameWithOwner string `json:"nameWithOwner"`
	URL           string `json:"url"`
	SSHURL        string `json:"sshUrl"`
	IsLocal       bool   `json:"isLocal"`
	LocalName     string `json:"localName"`
} // @name remote.Project

type CloneRequest struct {
	NameWithOwner      string
	ProjectDirectories []string
	DirectoryIndex     int
	LocalProjectNames  []string
}

type ClonedProject struct {
	Name string `json:"name"`
	Path string `json:"path"`
} // @name remote.ClonedProject
