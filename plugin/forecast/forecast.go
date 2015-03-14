// Package forecast is a bort IRC plugin that generates ascii forecasts using
// National Weather Service data.
package forecast

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"time"

	"github.com/ianremmler/bort"
	"github.com/jteeuwen/go-pkg-xmlx"
	"golang.org/x/net/html/charset"
)

const (
	fcURLFmt    = "http://forecast.weather.gov/MapClick.php?lat=%s&lon=%s&FcstType=digitalDWML"
	locURLFmt   = "http://nominatim.openstreetmap.org/search?format=json&limit=1&q=%s"
	arrows      = "↑↗→↘↓↙←↖"
	firstOctile = 0x2581
	maxHours    = 48
)

var (
	errFc  = errors.New("Error retrieving forecast.")
	errLoc = errors.New("I had a problem finding that location.")
)

type location struct {
	Name string `json:"display_name"`
	Lat  string `json:"lat"`
	Lon  string `json:"lon"`
}

// Forecast returns a pretty-printed forecast for the given location.  The
// location may be anything understood by OpenStreetMap's Nominatim service.
func Forecast(in, out *bort.Message) error {
	loc := regexp.MustCompile("\\s+").ReplaceAllLiteralString(in.Text, "+")
	outp, err := http.Get(fmt.Sprintf(locURLFmt, loc))
	if err != nil {
		return errLoc
	}
	defer outp.Body.Close()
	if outp.StatusCode != 200 {
		return errLoc
	}
	dec := json.NewDecoder(outp.Body)
	locs := []location{}
	dec.Decode(&locs)
	if len(locs) == 0 {
		return errLoc
	}
	fc, err := forecast(locs[0])
	if err != nil {
		return errFc
	}
	out.Text = fc
	return nil
}

func forecast(loc location) (string, error) {
	doc := xmlx.New()
	url := fmt.Sprintf(fcURLFmt, loc.Lat, loc.Lon)
	err := doc.LoadUri(url, func(str string, rdr io.Reader) (io.Reader, error) {
		return charset.NewReader(rdr, str)
	})
	if err != nil {
		return "", errFc
	}

	startTimeNodes := doc.SelectNodes("", "start-valid-time")
	endTimeNodes := doc.SelectNodes("", "end-valid-time")
	if len(startTimeNodes) == 0 || len(endTimeNodes) == 0 {
		return "", errFc
	}
	if len(endTimeNodes) > maxHours {
		endTimeNodes = endTimeNodes[:maxHours]
	}
	startTime, _ := time.Parse(time.RFC3339, startTimeNodes[0].GetValue())
	endTime, _ := time.Parse(time.RFC3339, endTimeNodes[len(endTimeNodes)-1].GetValue())

	temps := findVals("temperature", "hourly", doc)
	humids := findVals("humidity", "", doc)
	precips := findVals("probability-of-precipitation", "", doc)
	speeds := findVals("wind-speed", "sustained", doc)
	dirs := findVals("direction", "", doc)

	minTemp, maxTemp, tempGraph := makeGraph(temps)
	minHumid, maxHumid, humidGraph := makeGraph(humids)
	minPrecip, maxPrecip, precipGraph := makeGraph(precips)
	minSpeed, maxSpeed, speedGraph := makeGraph(speeds)

	dirGraph := ""
	for _, dir := range dirs {
		idx := dirIndex(dir)
		dirGraph += string([]rune(arrows)[idx])
	}

	timeFmt := "2006-01-02 15:04"
	start, end := startTime.Format(timeFmt), endTime.Format(timeFmt)

	tempRange := fmt.Sprintf("%3d %3d", minTemp, maxTemp)
	humidRange := fmt.Sprintf("%3d %3d", minHumid, maxHumid)
	precipRange := fmt.Sprintf("%3d %3d", minPrecip, maxPrecip)
	speedRange := fmt.Sprintf("%3d %3d", minSpeed, maxSpeed)

	out := fmt.Sprintf("Forecast for %s\n", loc.Name)
	out += fmt.Sprintf("         min max %-24s%24s\n", start, end)
	out += fmt.Sprintf("Temp °F  %7s %s\n", tempRange, tempGraph)
	out += fmt.Sprintf("Humid %%  %7s %s\n", humidRange, humidGraph) // esc % 2X for later fmt use
	out += fmt.Sprintf("Precip %% %7s %s\n", precipRange, precipGraph)
	out += fmt.Sprintf("Wind mph %7s %s\n", speedRange, speedGraph)
	out += fmt.Sprintf("Wind dir         %s\n", dirGraph)

	return out, nil
}

func minmax(vals []int) (int, int) {
	min, max := math.MaxInt32, math.MinInt32
	for _, v := range vals {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max
}

func rescale(val, min, max, bins int) int {
	if min >= max {
		return 0
	}
	v := (val - min) * bins / (max - min)
	if v < 0 {
		v = 0
	} else if v > bins-1 {
		v = bins - 1
	}
	return v
}

func dirIndex(dir int) int {
	return ((dir + 360/16) * 8 / 360) % 8
}

func findVals(name, typ string, doc *xmlx.Document) []int {
	vals := []int{}
	nodes := doc.SelectNodes("", name)
	for _, node := range nodes {
		if typ == "" || node.As("", "type") == typ {
			for _, kid := range node.Children {
				vals = append(vals, kid.I("", "value"))
				if len(vals) >= maxHours {
					break
				}
			}
			break // just use the first set
		}
	}
	return vals
}

func makeGraph(vals []int) (int, int, string) {
	if len(vals) == 0 {
		return 0, 0, ""
	}
	graph := ""
	min, max := minmax(vals)
	for _, val := range vals {
		octile := rescale(val, min, max, 8)
		graph += string(firstOctile + octile)
	}
	return min, max, graph
}

func init() {
	bort.RegisterCommand("forecast", "asciitastic 2 day NWS forecast for a given location", Forecast)
}
