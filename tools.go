package main

import "math"

func Round(x float64) float64 {
	return math.Round(x*100) / 100
}

func Mul(x, y float64) float64 {
	return Round(x * y)
}

func Div(x, y float64) float64 {
	return Round(x / y)
}
