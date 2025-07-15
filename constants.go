package main

// This is max acceptable length
// for a chirp
const (
	maxMsgLength = 140
)

// Response Types for Error
type ResponseError int

const (
	ServerError ResponseError = iota
	Rejected
)
