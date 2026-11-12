package chart

import (
	"bytes"
	"errors"
	"fmt"
	"image/color"
	"image/png"
	"time"

	"newsbot/internal/market"
	"newsbot/internal/news"
)

const (
	width       = 960
	height      = 460
	plotLeft    = 20
	plotRight   = 790
	plotTop     = 64
	plotBottom  = 330
	volumeTop   = 350
	volumeBot   = 440
	labelX      = 800
	titleScale  = 3
	labelScale  = 2
	lineWeight  = 3
	minBars     = 10
	areaOpacity = 45
)

var (
	background = color.RGBA{R: 15, G: 20, B: 25, A: 255}
	gridColor  = color.RGBA{R: 255, G: 255, B: 255, A: 18}
	textColor  = color.RGBA{R: 225, G: 230, B: 235, A: 255}
	mutedColor = color.RGBA{R: 140, G: 150, B: 160, A: 255}
	volumeGray = color.RGBA{R: 150, G: 160, B: 170, A: 170}
	trendColor = map[news.Direction]color.RGBA{
		news.Up:    {R: 34, G: 197, B: 94, A: 255},
		news.Down:  {R: 239, G: 68, B: 68, A: 255},
		news.Mixed: {R: 245, G: 158, B: 11, A: 255},
	}
)

type Spec struct {
	Contract string
	Interval string
	Bars     []market.Bar
	Marker   time.Time
	Trend    news.Direction
}

func Render(spec Spec) ([]byte, error) {
	if len(spec.Bars) < minBars {
		return nil, errors.New("chart: not enough bars")
	}

	accent, ok := trendColor[spec.Trend]
	if !ok {
		accent = trendColor[news.Mixed]
	}

	c := newCanvas(width, height, background)
	frame := newFrame(spec.Bars)

	c.drawGrid()
	c.drawArea(frame, withAlpha(accent, areaOpacity))
	c.drawPrice(frame, accent)
	c.drawVolume(frame, spec.Marker, accent)
	c.drawMarker(frame, spec.Marker)
	c.drawLabels(frame, spec, accent)

	var buf bytes.Buffer

	if err := png.Encode(&buf, c.img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type frame struct {
	bars      []market.Bar
	low, high float64
	maxVolume float64
}

func newFrame(bars []market.Bar) frame {
	f := frame{bars: bars, low: bars[0].Close, high: bars[0].Close}

	for _, b := range bars {
		f.low = min(f.low, b.Close)
		f.high = max(f.high, b.Close)
		f.maxVolume = max(f.maxVolume, b.QuoteVolume)
	}

	if f.high == f.low {
		f.high *= 1.001
		f.low *= 0.999
	}

	return f
}

func (f frame) x(i int) int {
	return plotLeft + i*(plotRight-plotLeft)/(len(f.bars)-1)
}

func (f frame) y(price float64) int {
	return plotBottom - int((price-f.low)/(f.high-f.low)*float64(plotBottom-plotTop))
}

func (f frame) indexAt(t time.Time) int {
	for i, b := range f.bars {
		if !b.Open.Before(t) {
			return i
		}
	}

	return len(f.bars) - 1
}

func (c canvas) drawGrid() {
	for k := 0; k <= 4; k++ {
		y := plotTop + k*(plotBottom-plotTop)/4
		c.rect(plotLeft, y, plotRight, y+1, gridColor)
	}
}

func (c canvas) drawArea(f frame, fill color.RGBA) {
	for i := 0; i < len(f.bars)-1; i++ {
		top := min(f.y(f.bars[i].Close), f.y(f.bars[i+1].Close))
		c.rect(f.x(i), top, f.x(i+1), plotBottom, fill)
	}
}

func (c canvas) drawPrice(f frame, accent color.RGBA) {
	for i := 0; i < len(f.bars)-1; i++ {
		c.line(f.x(i), f.y(f.bars[i].Close), f.x(i+1), f.y(f.bars[i+1].Close), lineWeight, accent)
	}
}

func (c canvas) drawVolume(f frame, marker time.Time, accent color.RGBA) {
	if f.maxVolume <= 0 {
		return
	}

	barWidth := max((plotRight-plotLeft)/len(f.bars)-1, 1)
	from := f.indexAt(marker)

	for i, b := range f.bars {
		h := int(b.QuoteVolume / f.maxVolume * float64(volumeBot-volumeTop))
		col := volumeGray

		if !marker.IsZero() && i >= from {
			col = withAlpha(accent, 200)
		}

		c.rect(f.x(i)-barWidth/2, volumeBot-h, f.x(i)-barWidth/2+barWidth, volumeBot, col)
	}
}

func (c canvas) drawMarker(f frame, marker time.Time) {
	if marker.IsZero() {
		return
	}

	i := f.indexAt(marker)
	x, y := f.x(i), f.y(f.bars[i].Close)

	c.dashedVertical(x, plotTop, volumeBot, textColor)
	c.rect(x-5, y-5, x+6, y+6, textColor)
}

func (c canvas) drawLabels(f frame, spec Spec, accent color.RGBA) {
	last := f.bars[len(f.bars)-1].Close
	first := f.bars[0].Close
	change := news.FormatPercent((last/first - 1) * 100)
	title := fmt.Sprintf("%s  %s", spec.Contract, spec.Interval)

	c.text(plotLeft, 18, titleScale, title, textColor)
	c.text(plotRight-textWidth(change, titleScale), 18, titleScale, change, accent)

	c.text(labelX, plotTop-4, labelScale, news.FormatPrice(f.high), mutedColor)
	c.text(labelX, plotBottom-10, labelScale, news.FormatPrice(f.low), mutedColor)

	y := f.y(last) - 7
	c.rect(labelX-4, y-4, width, y+18, accent)
	c.text(labelX, y, labelScale, news.FormatPrice(last), background)
}

func withAlpha(col color.RGBA, a uint8) color.RGBA {
	col.A = a

	return col
}
