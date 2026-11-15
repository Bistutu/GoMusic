package models

// Track retains original title and artists separately, including literal " - " in titles.
type Track struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Artists []string `json:"artists"`
}

// SongsCount is the exported count. TotalCount preserves the upstream count.
type SongList struct {
	Name       string   `json:"name"`
	Source     string   `json:"source"`
	Songs      []string `json:"songs"`
	Tracks     []Track  `json:"tracks"`
	SongsCount int      `json:"songs_count"`
	TotalCount int      `json:"total_count"`
	Complete   bool     `json:"complete"`
	Warnings   []string `json:"warnings"`
}
