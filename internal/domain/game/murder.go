package game

// Murder scenario loaded from JSON
type Murder struct {
	Title      string      `json:"title"`
	Killer     string      `json:"killer"`
	Weapon     string      `json:"weapon"`
	Location   string      `json:"location"`
	Characters []Character `json:"characters"`
}
