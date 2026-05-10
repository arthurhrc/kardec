// Chart demonstrates kardec/chart — bar, line, and pie charts
// rendered as SVG and embedded as vector Form XObjects.
//
//	go run ./examples/chart
//
// Produces chart.pdf with one section per chart type, each
// rendered from a small data slice (~5 LoC per chart).
package main

import (
	"log"

	"github.com/arthurhrc/kardec"
	"github.com/arthurhrc/kardec/chart"
	_ "github.com/arthurhrc/kardec/render"
)

func main() {
	doc := kardec.New(kardec.PageA4, kardec.MarginsNormal).
		Heading(1, kardec.Text("Charts"))

	// Bar chart.
	doc.Heading(2, kardec.Text("Q4 revenue (bar)"))
	bar := chart.Bar(chart.BarChart{
		Title:  "Q4 revenue by month",
		YLabel: "R$ thousands",
		Series: []chart.BarSeries{
			{Label: "Oct", Value: 12.5},
			{Label: "Nov", Value: 14.2},
			{Label: "Dec", Value: 11.8},
		},
	})
	doc.Image(bar.Render(400, 240)).Width(kardec.Pt(400)).Build()

	// Line chart with two series.
	doc.Heading(2, kardec.Text("Latency over time (line)"))
	line := chart.Line(chart.LineChart{
		Title:  "Request latency — last 6 minutes",
		XLabel: "minute",
		YLabel: "ms",
		Series: []chart.LineSeries{
			{Label: "p50", Points: []chart.Point{
				{X: 1, Y: 120}, {X: 2, Y: 130}, {X: 3, Y: 125},
				{X: 4, Y: 140}, {X: 5, Y: 138}, {X: 6, Y: 142},
			}},
			{Label: "p99", Points: []chart.Point{
				{X: 1, Y: 340}, {X: 2, Y: 380}, {X: 3, Y: 360},
				{X: 4, Y: 420}, {X: 5, Y: 410}, {X: 6, Y: 435},
			}},
		},
	})
	doc.Image(line.Render(450, 260)).Width(kardec.Pt(450)).Build()

	// Pie chart.
	doc.Heading(2, kardec.Text("Browser share (pie)"))
	pie := chart.Pie(chart.PieChart{
		Title: "Visits — Jan 2026",
		Slices: []chart.PieSlice{
			{Label: "Chrome", Value: 65},
			{Label: "Safari", Value: 18},
			{Label: "Firefox", Value: 12},
			{Label: "Other", Value: 5},
		},
	})
	doc.Image(pie.Render(420, 240)).Width(kardec.Pt(420)).Build()

	if err := doc.Render("chart.pdf"); err != nil {
		log.Fatalf("render: %v", err)
	}
	log.Println("rendered chart.pdf")
}
