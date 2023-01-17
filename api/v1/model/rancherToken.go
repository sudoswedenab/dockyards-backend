package model

type RRtoken struct {
	Name   string `json:"name"`
	UserId string `json:"userId"`
	Ttl    int    `json:"ttl"`
}
