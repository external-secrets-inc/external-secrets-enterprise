package github

type GithubFile struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"` // "file", "dir"
	DownloadURL string `json:"download_url"`
}

type GithubContentFile struct {
	Content  string `json:"content"` // base64 encoded
	SHA      string `json:"sha"`
	Path     string `json:"path"`
	Type     string `json:"type"`
	Encoding string `json:"encoding"`
}
