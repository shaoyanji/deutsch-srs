package backend

type AddCardInput struct {
	Deck     string
	Front    string
	Back     string
	NoteType string
	Tags     []string
}

type Stats struct {
	TotalCards      int
	DueToday        int
	NewCards        int
	ReviewsToday    int
	BackendName     string
	BackendHealthy  bool
	HealthErrorText string
}
