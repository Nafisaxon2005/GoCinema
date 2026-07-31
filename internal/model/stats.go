package model

type ShowSalesStat struct {
	ShowID  int64  `json:"show_id"`
	Title   string `json:"title"`
	Sold    int    `json:"sold"`
	Total   int    `json:"total"`
	Revenue int64  `json:"revenue"`
}
