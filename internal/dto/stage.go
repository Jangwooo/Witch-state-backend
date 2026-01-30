package dto

type StageResponse struct {
	ID         string `json:"id"`
	MusicID    string `json:"music_id"`
	LevelName  string `json:"level_name"`
	Difficulty int    `json:"difficulty"`
	TotalNotes int    `json:"total_notes"`
	MaxCombo   int    `json:"max_combo"`
}
