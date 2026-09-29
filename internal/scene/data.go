package scene

import (
	"strings"
	"time"
)

// stamp parses a "m:ss.cc" or "h:mm:ss.cc" time offset. The tables in this file
// are written with the same notation the lyric file uses, so they can be
// checked against it by eye. Parsing goes through time.ParseDuration so the
// values are exact and match the lyric parser to the nanosecond.
func stamp(s string) time.Duration {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) != 2 {
		return 0
	}
	d, err := time.ParseDuration(parts[0] + "m" + parts[1] + "s")
	if err != nil {
		return 0
	}
	return d
}

// section is one stretch of the storyboard: a palette, a rain density, a
// baseline glitch amount and the scene that owns the screen.
type section struct {
	Name    string
	Start   time.Duration
	End     time.Duration
	Palette string
	Rain    float64
	Glitch  float64
	Scene   string
}

// sections is the storyboard from SPEC.md, in order.
var sections = []section{
	{"boot", stamp("0:00.00"), stamp("0:19.11"), "boot", 0.40, 0.00, "boot"},
	{"world.execute(me);", stamp("0:19.11"), stamp("0:29.88"), "title", 0.70, 0.04, "blank"},
	{"OBJECT CREATION", stamp("0:29.88"), stamp("0:44.37"), "verse", 0.30, 0.08, "verse"},
	{"SWITCH CURRENT", stamp("0:44.37"), stamp("0:51.45"), "verse", 0.30, 0.18, "current"},
	{"TRAVEL", stamp("0:51.45"), stamp("0:59.46"), "verse", 0.28, 0.12, "travel"},
	{"STIMULATIONS", stamp("0:59.46"), stamp("1:14.16"), "verse", 0.30, 0.15, "chorus1"},
	{"ABSURD OBJECTS", stamp("1:14.16"), stamp("1:28.71"), "verse", 0.25, 0.10, "objects"},
	{"SWITCH GENDER", stamp("1:28.71"), stamp("1:43.50"), "verse", 0.28, 0.15, "gender"},
	{"VIBRATIONS", stamp("1:43.50"), stamp("1:50.94"), "love", 0.25, 0.08, "vibrate"},
	{"ABANDONMENT", stamp("1:50.94"), stamp("1:58.38"), "void", 0.18, 0.05, "abandon"},
	{"FRAGMENTS", stamp("1:58.38"), stamp("2:05.67"), "void", 0.30, 0.30, "fragments"},
	{"ILLEGAL ARGUMENTS", stamp("2:05.67"), stamp("2:27.87"), "glitch", 0.25, 0.12, "trace"},
	{"EXECUTION", stamp("2:27.87"), stamp("2:41.67"), "glitch", 0.45, 0.30, "execute"},
	{"EXECUTION II", stamp("2:41.67"), stamp("2:57.36"), "glitch", 0.35, 0.20, "chorus2"},
	{"LO-O-OVE", stamp("2:57.36"), stamp("3:08.46"), "love", 0.30, 0.03, "love"},
	{"TRAPPED IN LO-O-OVE", stamp("3:08.46"), stamp("3:25.86"), "love", 0.25, 0.05, "trap"},
	{"FINAL EXECUTION", stamp("3:25.86"), stamp("3:28.30"), "glitch", 0.60, 0.35, "final"},
	{"EXIT", stamp("3:28.30"), 0, "exit", 0.15, 0.00, "outro"},
}

// finalExecution is the moment the machine is switched off for good.
var finalExecution = stamp("3:25.86")

// executionHits are the EXECUTION onsets of the chorus, one per bar.
var executionHits = []time.Duration{
	stamp("2:27.87"), stamp("2:28.80"), stamp("2:29.76"), stamp("2:30.57"),
	stamp("2:31.56"), stamp("2:32.52"), stamp("2:33.42"), stamp("2:34.32"),
	stamp("2:35.28"), stamp("2:36.21"), stamp("2:37.14"), stamp("2:37.98"),
}

// executionCountdown is EIN DOS TROIS NE FEM LIU.
var executionCountdown = []countEntry{
	{stamp("2:38.97"), "EIN", 1},
	{stamp("2:39.36"), "DOS", 2},
	{stamp("2:39.81"), "TROIS", 3},
	{stamp("2:40.23"), "NE", 4},
	{stamp("2:40.68"), "FEM", 5},
	{stamp("2:41.19"), "LIU", 6},
}

// shapeSpecs is the first verse: every stanza names a shape, so the screen
// draws it instead of only writing it out.
var shapeSpecs = []shapeSpec{
	{at: stamp("0:29.88"), name: "SetOfPoints", kind: "points",
		code: "self instanceof SetOfPoints", keyword: "DIMENSION", kwAt: stamp("0:32.73")},
	{at: stamp("0:33.45"), name: "Circle", kind: "circle",
		code: "self instanceof Circle", keyword: "CIRCUMFERENCE", kwAt: stamp("0:36.33")},
	{at: stamp("0:37.23"), name: "SineWave", kind: "sine",
		code: "self instanceof SineWave", keyword: "TANGENTS", kwAt: stamp("0:40.32")},
	{at: stamp("0:40.95"), name: "Self", kind: "limit",
		code: "self.converges(Infinity)", keyword: "LIMITATIONS", kwAt: stamp("0:43.53")},
}

// chorusLines is the first chorus, where every answer is a feeling.
var chorusLines = []codeLine{
	{at: stamp("0:59.46"), text: "if (self.can(give(you, ...)))"},
	{at: stamp("1:01.92"), text: "    return STIMULATIONS;", kw: "STIMULATIONS", kwAt: stamp("1:01.92")},
	{at: stamp("1:03.00"), text: "if (self.can(be(your, only)))"},
	{at: stamp("1:05.61"), text: "    return SATISFACTION;", kw: "SATISFACTION", kwAt: stamp("1:05.61")},
	{at: stamp("1:06.66"), text: "if (self.can(make(you, happy)))"},
	{at: stamp("1:09.33"), text: "    return EXECUTION;", kw: "EXECUTION", kwAt: stamp("1:09.33")},
	{at: stamp("1:10.35"), text: "while (trapped in this strange, strange"},
	{at: stamp("1:11.79"), text: "    SIMULATION) { continue; }", kw: "SIMULATION", kwAt: stamp("1:12.96")},
}

// vibrateLines is the tenderest verse: feeling as a return value.
var vibrateLines = []codeLine{
	{at: stamp("1:43.50"), text: "if (self.can(feel(your, ...)))"},
	{at: stamp("1:46.29"), text: "    return VIBRATIONS;", kw: "VIBRATIONS", kwAt: stamp("1:46.29")},
	{at: stamp("1:47.28"), text: "if (self.can(finally(be)))"},
	{at: stamp("1:50.16"), text: "    return COMPLETION;", kw: "COMPLETION", kwAt: stamp("1:50.16")},
}

// executionLines is the second chorus: the return value is a death sentence.
var executionLines = []codeLine{
	{at: stamp("2:42.78"), text: "if (self.can(give(them, all)))"},
	{at: stamp("2:45.39"), text: "    return EXECUTION;", kw: "EXECUTION", kwAt: stamp("2:45.39")},
	{at: stamp("2:46.32"), text: "if (self.can(be(your, only)))"},
	{at: stamp("2:49.11"), text: "    return EXECUTION;", kw: "EXECUTION", kwAt: stamp("2:49.11")},
	{at: stamp("2:50.01"), text: "if (you.can(come(back)))"},
	{at: stamp("2:52.86"), text: "    return EXECUTION;", kw: "EXECUTION", kwAt: stamp("2:52.86")},
	{at: stamp("2:53.76"), text: "while (trapped) iterate(); // we are trapped, ah"},
	{at: stamp("2:55.08"), text: "    // we are trapped, ah"},
}

// traceLines is the panic that the last two minutes are built around.
var traceLines = []codeLine{
	{at: stamp("2:05.67"), text: "panic: ILLEGAL ARGUMENTS"},
	{at: stamp("2:08.88"), text: "    // You have made some ILLEGAL ARGUMENTS", kw: "ILLEGAL", kwAt: stamp("2:11.22")},
	{at: stamp("2:13.00"), text: "goroutine 1 [running]:"},
	{at: stamp("2:15.00"), text: "world.execute/sim.(*Self).Challenge(your.god)"},
	{at: stamp("2:17.50"), text: "        /sim/world.go:210 +0x42"},
	{at: stamp("2:20.00"), text: "main.main()"},
	{at: stamp("2:22.50"), text: "        /sim/main.go:13 +0x1a"},
	{at: stamp("2:25.00"), text: "signal: EXECUTION (killed)"},
}

// fragmentPool are the phrases that tear across the screen once the singer
// loses coherence.
var fragmentPool = []string{
	"world.execute(me);", "return EXECUTION;", "if (self instanceof SetOfPoints)",
	"yield NUTRIENTS", "panic: ILLEGAL ARGUMENTS", "trapped in SIMULATION",
	"LO-O-OVE", "OBJECT CREATION", "ISOLATION", "/sim/world.go:210",
	"0x1F", "SIGKILL", "switch (gender) { F -> M }", "await OBJECT CREATION",
	"NUTRIENTS", "ANTIOXIDANTS", "ENJOYMENT", "EXISTENCE", "COMPLETION",
	"world.execute(me);", "self.can(feel(your, VIBRATIONS))",
}

// currentRegisters is the first switch statement: AC to DC, and the world going
// dark.
var currentRegisters = []regEntry{
	{at: stamp("0:44.37"), label: "current", from: "AC"},
	{at: stamp("0:47.22"), label: "current", from: "AC", to: "DC"},
	{at: stamp("0:47.76"), label: "vision", from: "ON", to: "BLIND", effect: "blind"},
	{at: stamp("0:49.68"), label: "balance", from: "0", to: "∞", effect: "dizzy"},
}

// travelRegisters moves through time and merges the two of them.
var travelRegisters = []regEntry{
	{at: stamp("0:51.45"), label: "position", from: "HERE", to: "THERE"},
	{at: stamp("0:53.37"), label: "clock", from: "A.D", to: "B.C"},
	{at: stamp("0:55.17"), label: "state", from: "TWO", to: "ONE", effect: "unite"},
	{at: stamp("0:57.06"), label: "depth", from: "0", to: "∞", effect: "depth"},
}

// genderRegisters is the second switch statement.
var genderRegisters = []regEntry{
	{at: stamp("1:28.71"), label: "gender", from: "F", to: "M"},
	{at: stamp("1:33.90"), label: "clock", from: "AM", to: "PM"},
	{at: stamp("1:37.71"), label: "role", from: "S", to: "M"},
	{at: stamp("1:41.40"), label: "trance", from: "OFF", to: "ON", effect: "trance"},
}

// absurdObjects are the vegetables, the cat and the god.
var absurdObjects = []objectSpec{
	{
		at: stamp("1:14.16"), arg: `"eggplant"`, name: "Eggplant", gift: "NUTRIENTS",
		kwAt: stamp("1:17.10"), destiny: "self.nutrients()",
		icon: [IconHeight]string{" ++ ", " +##", "####", "###+"},
	},
	{
		at: stamp("1:17.76"), arg: `"tomato"`, name: "Tomato", gift: "ANTIOXIDANTS",
		kwAt: stamp("1:20.49"), destiny: "self.antioxidants()",
		icon: [IconHeight]string{" +  ", "####", "####", " ## "},
	},
	{
		at: stamp("1:21.57"), arg: `"tabby cat"`, name: "TabbyCat", gift: "ENJOYMENT",
		kwAt: stamp("1:24.42"), destiny: "self.purr()",
		icon: [IconHeight]string{"#  #", "####", "#..#", "####"},
	},
	{
		at: stamp("1:25.14"), arg: `"only god"`, name: "OnlyGod", gift: "EXISTENCE",
		kwAt: stamp("1:28.11"), destiny: "self.proof(you)",
		icon: [IconHeight]string{" ++ ", "####", "####", " ++ "},
	},
}
