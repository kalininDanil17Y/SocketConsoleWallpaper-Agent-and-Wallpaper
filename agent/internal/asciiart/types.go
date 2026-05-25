package asciiart

type Frame struct {
	Type    string   `json:"type"`
	Width   int      `json:"w"`
	Height  int      `json:"h"`
	Palette []string `json:"palette"`
	Rows    []Row    `json:"rows"`
	Source  string   `json:"source,omitempty"`
}

type Row struct {
	Text string `json:"text"`
	FG   []int  `json:"fg"`
}

type ImageInfo struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	ModTime   string `json:"modTime"`
	SizeBytes int64  `json:"sizeBytes"`
}
