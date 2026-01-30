package dto

type MusicResponse struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Artist        string  `json:"artist"`
	Composer      string  `json:"composer"`
	Bpm           float64 `json:"bpm"`
	Genre         string  `json:"genre"`
	Description   string  `json:"description"`
	IsRecommended bool    `json:"is_recommended"`
	IsFree        bool    `json:"is_free"`
	UnlockLevel   int     `json:"unlock_level"`
	ReleaseDate   *string `json:"release_date"`
}
