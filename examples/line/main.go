package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	ws "github.com/giannimassi/go-echarts-ws"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
)

var days = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

func main() {
	// create a new line instance
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeChalk}),
		charts.WithTitleOpts(opts.Title{Title: "Line example"}))

	// Put data into instance
	values1, values2 := generateRandValues(7, 300), generateRandValues(7, 3000)
	line.SetXAxis(days).
		AddSeries("Category A", generateLineItems(values1)).
		SetSeriesOptions(charts.WithLineChartOpts(opts.LineChart{Smooth: true}))

	// Setup handler and obtains data that we're going to be using
	// to pass updates to be written over ws.
	wsHandler, dataC := ws.Handler()
	defer close(dataC)
	updateChart := func() {
		days = append(days[:], days[0])
		values1 = append(values1[:], rand.Intn(300))
		values2 = append(values2[:], rand.Intn(300))
		line.MultiSeries = line.MultiSeries[:0]
		line.SetXAxis(days).
			AddSeries("Category A", generateLineItems(values1))
		line.Validate()
		dataC <- line.JSON()
	}

	go func() {
		for {
			select {
			case <-time.After(time.Second):
				updateChart()
			}
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { ws.Render(w, line, line.ChartID, r.Host) })
	http.HandleFunc("/ws", wsHandler)
	log.Println("Visit http://localhost:8080 to see the live chart")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func generateRandValues(len, max int) (values []int) {
	for i := 0; i < len; i++ {
		values = append(values, rand.Intn(max))
	}
	return
}

// generate random data for line chart
func generateLineItems(values []int) []opts.LineData {
	items := make([]opts.LineData, 0)
	for _, v := range values {
		items = append(items, opts.LineData{Value: v})
	}
	return items
}
