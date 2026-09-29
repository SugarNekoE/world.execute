package scene

import (
	"fmt"
	"math"
	"strings"
	"time"

	"world.execute/internal/term"
)

type visualCue struct {
	at          time.Duration
	kind, label string
}

var lyricVisuals = map[string][]visualCue{
	"boot": {
		{stamp("0:00.00"), "power", "POWER LINE / connecting"},
		{stamp("0:03.12"), "shield", "PROTECTION / shield online"},
		{stamp("0:06.42"), "assemble", "OBJECT CREATION / assembling"},
		{stamp("0:10.32"), "assemble", "INITIALIZATION / loading parameters"},
		{stamp("0:13.89"), "world", "SIMULATION / world online"},
	},
	"chorus1": {
		{stamp("0:59.46"), "network", "STIMULATIONS / signal propagation"},
		{stamp("1:05.61"), "satisfy", "SATISFACTION / target acquired"},
		{stamp("1:09.33"), "execute", "EXECUTION / instruction fired"},
		{stamp("1:10.35"), "cage", "SIMULATION / no way out"},
	},
	"vibrate": {
		{stamp("1:43.50"), "vibrate", "VIBRATIONS / seeking resonance"},
		{stamp("1:50.16"), "complete", "COMPLETION / in phase"},
	},
	"abandon":   {{stamp("1:50.94"), "isolate", "CONNECTION LOST / isolation"}},
	"fragments": {{stamp("1:58.38"), "fragments", "FRAGMENTS / erasing the heart"}},
	"trace": {
		{stamp("2:05.67"), "judge", "CHALLENGING YOUR GOD / judgment"},
		{stamp("2:11.22"), "error", "ILLEGAL ARGUMENTS / rejected"},
	},
	"chorus2": {
		{stamp("2:41.67"), "execute", "EXECUTION / all targets"},
		{stamp("2:45.39"), "execute", "EXECUTION / all targets"},
		{stamp("2:49.11"), "execute", "EXECUTION / only target"},
		{stamp("2:52.86"), "execute", "EXECUTION / return to me"},
		{stamp("2:53.76"), "cage", "TRAPPED / recursive simulation"},
	},
}

type IllustratedScene struct {
	stage Layer
	cues  []visualCue
}

func (s *IllustratedScene) Draw(ctx *Context) {
	area := ctx.Area
	if area.H < 15 || area.W-ctx.SideW < 34 {
		s.stage.Draw(ctx)
		return
	}
	top := *ctx
	top.Area.H = max(6, area.H/3)
	s.stage.Draw(&top)
	r := Rect{X: area.X + 3, Y: area.Y + top.Area.H + 1, W: area.W - ctx.SideW - 6, H: area.H - top.Area.H - 2}
	ctx.Panel(r)
	cue := s.cues[0]
	for _, next := range s.cues {
		if next.at <= ctx.T {
			cue = next
		}
	}
	motionLabel(ctx, r, cue.label)
	r.Y++
	r.H--
	c := motionCanvas(ctx, r)
	t := ctx.Since(cue.at).Seconds()
	drawIllustration(ctx, c, cue.kind, t)
	c.Flush()
	drawIllustrationArt(ctx, r, cue, t)
	if cue.kind == "isolate" {
		drawPingLog(ctx, r, cue)
	}
}

// drawPingLog lists the pings sent to the one who left, each timing out, and a
// packet loss gauge that climbs to a hundred percent.
func drawPingLog(ctx *Context, r Rect, cue visualCue) {
	if r.W < 40 || r.H < 6 {
		return
	}
	pal := ctx.Palette
	s := ctx.Screen
	t := ctx.Since(cue.at).Seconds()
	seq := int(t * 1.6)
	lines := min(4, r.H-2)
	for i := range lines {
		n := seq - i
		if n < 0 {
			break
		}
		fg := Fade(pal.Err, pal.Shadow, 1-0.22*float64(i))
		s.Text(r.X, r.Bottom()-1-i, fmt.Sprintf("ping you  seq=%02d  timeout", n), fg, term.ColorDefault, term.Attr(0))
	}
	loss := Progress(ctx.T, cue.at, stamp("1:57.51"))
	const bar = 10
	filled := int(loss * bar)
	label := fmt.Sprintf("packet loss %3d%%", int(loss*100))
	x := r.Right() - bar - 1 - term.StringWidth(label) - 1
	s.Text(x, r.Y, label, pal.Dim, term.ColorDefault, term.Attr(0))
	s.HLine(x+term.StringWidth(label)+1, r.Y, filled, '█', pal.Err, term.ColorDefault, term.Attr(0))
	s.HLine(x+term.StringWidth(label)+1+filled, r.Y, bar-filled, '░', pal.Dim, term.ColorDefault, term.Attr(0))
}

func drawIllustration(ctx *Context, c *Canvas, kind string, t float64) {
	switch kind {
	case "network", "isolate":
		remaining := 8
		if kind == "isolate" {
			remaining = max(0, 8-int(t*1.25))
		}
		for i := range 8 {
			a := float64(i)*math.Pi/4 + t*0.12
			x, y := 0.5+0.39*math.Cos(a), 0.5+0.40*math.Sin(a)
			if i >= remaining {
				continue
			}
			c.Line(0.5, 0.5, x, y, 0)
			p := math.Mod(t*0.9+float64(i)/8, 1)
			c.Disc(0.5+(x-0.5)*p, 0.5+(y-0.5)*p, 0.014, 1.4, 1)
		}
	case "satisfy":
		for i := range 5 {
			r := 0.09 + float64(i)*0.07
			c.Arc(0.5, 0.5, r, 1.05, t+float64(i), t+float64(i)+math.Pi*1.7, 1+i%2)
		}
		c.Polyline([]Point{{0.42, 0.5}, {0.48, 0.59}, {0.6, 0.37}}, 2)
	case "vibrate", "complete":
		align := Progress(ctx.T, stamp("1:47.28"), stamp("1:50.16"))
		for i := range 2 {
			c.Func(func(x float64) float64 {
				return 0.5 + 0.27*math.Sin(x*math.Pi*6-t*5+float64(i)*(1-align)*math.Pi)
			}, 1+i, 500)
		}
		if kind == "complete" {
			c.Circle(0.5, 0.5, 0.32, 1.2, 2)
			c.Polyline([]Point{{0.40, 0.48}, {0.48, 0.60}, {0.64, 0.33}}, 2)
		}
	case "error":
		c.Arc(0.5, 0.5, 0.46, 1, t, t+1.3, 3)
	case "execute":
		DrawBurst(c, math.Mod(t*1.6, 1))
		c.Line(0.5, 0.05, 0.5, 0.95, 2)
		c.Line(0.08, 0.5, 0.92, 0.5, 2)
		c.Circle(0.5, 0.5, 0.12+0.04*math.Sin(t*5), 1.5, 1)
	}
}

func drawIllustrationArt(ctx *Context, r Rect, cue visualCue, t float64) {
	if r.W < 30 || r.H < 8 {
		return
	}
	pal := ctx.Palette
	art := r
	art.H = r.H - 1
	cx, cy := float64(art.W)/2, float64(art.H)
	R := math.Min(float64(art.W)*0.20, float64(art.H)*2*0.40)
	beat := float64(ctx.Energy)
	var box Rect
	title := ""
	var rows []hudRow
	progress := math.Min(t/6, 1)
	switch cue.kind {
	case "power":
		box, rows = drawPowerArt(ctx, art, t)
		title = "POWER LINE :: CONNECT"
		progress = math.Min(t/3.1, 1)
	case "world":
		box = globeArt(pal, cx, cy, R, t).draw(ctx, art, t)
		title = "SIMULATION :: WORLD"
		rows = []hudRow{{"LAT", fmt.Sprintf("%+06.2f", 40*math.Sin(t*0.7))}, {"LON", fmt.Sprintf("%+07.2f", math.Mod(t*28, 360)-180)}, {"TICK", fmt.Sprintf("%06d", int(t*60))}, {"STATE", "ONLINE"}}
	case "assemble":
		grow := ease(math.Min(t/1.6, 1))
		shape := boxArt(pal, cx, cy, R*1.5*grow, R*0.85*grow, 113)
		shape.feature = labelFeature(fmt.Sprintf("[ alloc 0x%04X ]", int(t*997)&0xffff), cx, cy, pal.Ink())
		box = shape.draw(ctx, art, t)
		title = "MEMORY :: ALLOCATE"
		rows = []hudRow{{"SIZE", fmt.Sprintf("%d KB", int(64*grow))}, {"PAGE", "RW-"}, {"OWNER", "self"}}
	case "cage":
		cageArt(pal, cx, cy, R*1.05, t).draw(ctx, art, t)
		box = heartArt(pal, cx, cy, R*0.62*(1+0.05*beat), pal.Accent).draw(ctx, art, t)
		title = "CONTAINMENT :: SIMULATION"
		rows = []hudRow{{"EXIT", "NONE"}, {"BARS", "ARMED"}, {"LOOP", "RECURSIVE"}}
	case "shield":
		box = shieldArt(pal, cx, cy, R).draw(ctx, art, t)
		title = "PROTECTION :: SHIELD"
		rows = []hudRow{{"INSUL", "OK"}, {"GND", "OK"}, {"LOAD", fmt.Sprintf("%02d%%", int(60+30*math.Sin(t*2)))}}
	case "error":
		box = warningArt(pal, cx, cy, R).draw(ctx, art, t)
		title = "EXCEPTION :: REJECTED"
		rows = []hudRow{{"CODE", "0xE1"}, {"ARG", "ILLEGAL"}, {"TRACE", "DUMPED"}}
	case "network", "isolate":
		remaining := 8
		if cue.kind == "isolate" {
			remaining = max(0, 8-int(t*1.25))
		}
		for i := range remaining {
			a := float64(i)*math.Pi/4 + t*0.12
			px := art.X + int(float64(art.W)*(0.5+0.39*math.Cos(a)))
			py := art.Y + int(float64(art.H)*(0.5+0.40*math.Sin(a)))
			col := pal.Accent2
			if i%2 == 1 {
				col = pal.Accent
			}
			ctx.Screen.Text(px-2, py, fmt.Sprintf("[N%d]", i), col, term.ColorDefault, term.Bold)
		}
		box = heartArt(pal, cx, cy, R*0.55*(1+0.05*beat), pal.Accent).draw(ctx, art, t)
		title = "LINK :: " + strings.ToUpper(cue.kind)
		rows = []hudRow{{"NODES", fmt.Sprintf("%d/8", remaining)}, {"LOSS", fmt.Sprintf("%d%%", (8-remaining)*12)}}
		progress = float64(remaining) / 8
	case "fragments":
		box, rows = drawFragmentsArt(ctx, art, t)
		title = "PURGE :: HEART"
		progress = Progress(ctx.T, stamp("1:58.38"), stamp("2:04.98"))
	case "judge":
		box, rows = drawJudgeArt(ctx, art, t)
		title = "JUDGMENT :: CHALLENGING YOUR GOD"
	default:
		return
	}
	drawHUD(ctx, art, box, title, rows, progress, t)
}
