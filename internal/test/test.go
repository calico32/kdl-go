package test

import "embed"

//go:embed kdl1/tests/test_cases/*
var Kdl1Tests embed.FS

//go:embed kdl2/tests/test_cases/*
var Kdl2Tests embed.FS
