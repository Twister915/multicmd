package main

type Task struct {
	I             int
	CommandFormat func(string) []string
	Target        string
}
