package main

// This is max acceptable length
// for a chirp
const (
	maxMsgLength = 140
	CHIRPY = "chirpy"
)

// Response Types for Error
type ResponseError int

const (
	ServerError ResponseError = iota
	Rejected
	NotFound
	Unauthorized
)
