# SPEC — `world.execute(me);` terminal animation

A program that plays Mili's *world.execute (me) ;* and renders a synchronised,
animated lyric video **inside the terminal**, written in Go.

This document is the authoritative record of what the program must do. It is
updated as work lands. Sections marked `[x]` are implemented, `[ ]` are not.

---

## 1. Goal

Turn the song into a terminal-native animation:

- the **audio** is the bundled MP3;
- the **animation** is produced as ANSI text/color frames, not pixels;
- the **lyrics** drive the visuals, using the word level timings already
  embedded in the file;
- the **viewer can control playback** from the keyboard, with a progress bar.

The tone matches the song: an artificial intelligence being switched on,
questioned, loved, and finally executed. The visuals therefore take the form of
a program's own log — boot sequence, code, pseudocode love, and a fatal
execution.

## 2. Decisions taken

| Question | Decision |
| --- | --- |
| Audio | Play the bundled MP3 in sync via an external player; fall back to a silent clock when no player is installed. |
| Visual style | Code/terminal aesthetic: boot log, falling code rain, pseudocode, glitch, block banners. No pixel framebuffer. |
| Lyric language | English only on screen. The Chinese translation stays in the asset file and is reachable with `--lang`. |
| Asset location | `assets/world-execute-me.mp3`, committed to the repository. |

## 3. Constraints

- Go only. Standard library plus `golang.org/x/term` for terminal handling.
- No generated binary assets committed; the envelope is computed at run time.
- Degrade instead of failing: no `mpv`, no `ffmpeg`, no colour, or a pipe
  instead of a TTY must all still produce something reasonable.
- Every file is `gofmt`ed, `go vet` clean, and unit tested where logic exists.

## 4. Module layout

Module path is `world.execute`, following standard Go project layout: an
`internal/` tree that nothing outside can import, and a single `main` under
`cmd/`.

```
world.execute/
├── cmd/world.execute/main.go     entry point, flag parsing, wiring, run loop
├── internal/
│   ├── config/      resolved run configuration and defaults
│   ├── lyric/       LRC parsing, word timings, timeline lookup
│   ├── audio/       player process control, clock, spectrum envelope
│   ├── term/        colour, screen buffer, banner font, raw mode
│   ├── control/     keymap and playback actions
│   └── scene/       the animation itself: layers, director, transport bar
├── assets/          the MP3 and the extracted LRC
└── SPEC.md, README.md, Makefile
```

## 5. Assets

- `assets/world-execute-me.mp3` — 211.906 s, 44.1 kHz stereo, 320 kbps.
- `assets/lyrics.lrc` — extracted from the MP3's `lyrics-XXX` tag. Enhanced
  LRC: `[line time]<word time>word …` for English, plus the Chinese line
  carrying the same line timestamp. The parser pairs them.

## 6. Storyboard

All times are offsets into the track.

| Time | Section | Visual |
| --- | --- | --- |
| 0:00.00 | Switch on the power line | Terminal boots. Log lines type themselves out, each with an `[ OK ]` stamp. |
| 0:03.12 | PROTECTION | The capitalised lyric surfaces as a highlighted token in the log. |
| 0:06.42 | OBJECT CREATION | `> allocating … object` with a progress spinner. |
| 0:10.32 | INITIALIZATION | Status counters fill in; the code rain accelerates. |
| 0:19.11 | world. | The title banner begins: `WORLD` appears and holds. |
| 0:29.25 | execute(me); | The rest of the banner types out in ~0.6 s, then holds. |
| 0:29.88 | Verse 1 | Each stanza becomes pseudocode: `if (self instanceof SetOfPoints) return DIMENSION;` with the capitalised word glowing as it is sung. |
| 0:44.37 | Switch my current | A switch/register panel: AC→DC, then the screen goes dark and smeared ("blind my vision"). |
| 1:06.66 | EXECUTION | The word lands hard for the first time: red flash, glitch. |
| 1:14.16 | eggplant / tomato / cat / god | Absurd objects are instantiated one per line, each with its own icon row. |
| 1:28.71 | Switch my gender | Register panel again: F→M, AM→PM, S→M. |
| 1:43.50 | VIBRATIONS | Waveform layer becomes prominent, reacting to the spectrum. |
| 1:50.94 | You have left | The log starts losing lines; a deletion counter climbs. |
| 1:57.51 | ISOLATION | The screen empties to a single blinking cursor. |
| 1:58.38 | FRAGMENTS | Fragments of earlier frames tear across the screen. |
| 2:05.67 | ILLEGAL ARGUMENTS | A stack trace appears, red, with the lyric as the exception message. |
| 2:27.87 | EXECUTION ×12 | Full screen announcement, one per line, each louder than the last. |
| 2:38.97 | EIN DOS TROIS NE FEM LIU | Countdown 1–6 across the screen. |
| 2:41.67 | EXECUTION | The chorus pseudocode repeats with the word as the return value. |
| 2:57.36 | LO-O-OVE | The only warmth in the piece: letters drift together, a soft palette. |
| 3:08.46 | Though you are free | Split screen: one half free, one half trapped. |
| 3:25.86 | EXECUTION (final) | The last flash. |
| 3:27.50 | Shutdown | Process teardown log, `exit code 0`, cursor restored. |

Layer composition, back to front:

1. **Rain** — falling code tokens; density and speed follow the energy envelope.
2. **Stage** — the section-specific visual listed above.
3. **Ambient** — a waveform wall along the bottom, a starfield, a shimmer grid
   and a pulse on the beat. It only paints into cells nothing else has claimed
   and never next to a word, so it fills the screen without ever covering text.
4. **Telemetry** — a system monitor for the boot and shutdown logs, a scrolling
   memory dump for the panic and the abandonment, in a strip the stage leaves
   free (`Context.SideW`).
5. **Chrome** — the header oscilloscope and the transport.
6. **Glitch** — short tear/noise bursts on transients and on key words.

**Transitions.** The last 300 ms of a section are a slanted wipe that ends
exactly on the boundary: the next section reveals from the left behind a `░▒▓`
edge while the current frame clears to the right, so the new section is
complete on the beat it belongs to. The impact sections (`execute`, `chorus2`,
`final`, `outro`) cut hard instead, because their first frame is the hit.

## 7. Playback controls

The transport is part of the animation, not a separate mode. The bottom of the
screen always carries a progress bar; key hints fade out after five seconds of
inactivity and return on any keypress.

```
▶  01:23.45 ━━━━━━━━━━━━●─────────────────────────  03:31.90   vol 80%  ⏸  60fps
space pause   ← → seek   ↑ ↓ volume   m mute   r restart   ? help   q quit
```

| Key | Action |
| --- | --- |
| `space`, `p` | pause / resume |
| `→`, `l` | seek forward 5 s |
| `←`, `h` | seek back 5 s |
| `.`, `,` | seek forward / back 1 s |
| `0`, `Home` | restart from the beginning |
| `End` | jump to the last second |
| `↑`, `+` | volume up 5 |
| `↓`, `-` | volume down 5 |
| `m` | mute / unmute |
| `[` `]` | shift the animation 10 ms later / earlier against the audio |
| `{` `}` | the same in 50 ms steps |
| `r` | restart the animation and audio from `--start` |
| `s` | toggle the info rows |
| `?` | help overlay |
| `q`, `Esc`, `Ctrl-C` | quit and restore the terminal |

Seeking and volume are applied to the audio player over mpv's JSON IPC when
mpv was selected. With ffplay, seeking restarts the process at the new offset;
volume changes are unavailable and the bar says so. With the silent backend the
clock alone moves.

Bar behaviour details:

- draws elapsed and total as `mm:ss.cc`, plus remaining time when the terminal
  is wide enough;
- the filled portion is drawn in the palette's accent colour, the handle is a
  single cell, and the track is a dim rule;
- when paused, the bar blinks and the play glyph becomes `⏸`;
- the bar occupies the last row, so scenes are clipped to `height-2`;
- at terminals narrower than 40 columns it collapses to `mm:ss / mm:ss`.

## 8. Flags

| Flag | Default | Meaning |
| --- | --- | --- |
| `--audio` | `assets/world-execute-me.mp3` | track to play |
| `--lyrics` | `assets/lyrics.lrc` | lyric file |
| `--player` | `auto` | `auto`, `mpv`, `ffplay`, `silent` |
| `--lang` | `en` | `en`, `zh`, `both` |
| `--fps` | `60` | animation frame rate |
| `--color` | `auto` | `auto`, `truecolor`, `256`, `16`, `none` |
| `--theme` | `dark` | `dark` or `light`; body text inherits the terminal's foreground |
| `--start` | `0` | start offset |
| `--duration` | full | stop after this long |
| `--delay` | saved value, else `0` | shifts the animation against the audio; positive shows it earlier, negative later. `[` `]` adjust it live and the value is remembered |
| `--volume` | `80` | initial volume |
| `--seed` | random | code rain seed |
| `--no-analysis` | `false` | skip the ffmpeg envelope |
| `--width`, `--height` | terminal | force the cell grid, useful for recording |
| `--frames` | `0` | render this many frames as fast as possible |
| `--record` | — | write frames to a file instead of a TTY |
| `--quiet` | `false` | suppress the startup log |
| `--verbose` | `false` | let the audio player log to stderr |

`--start`, `--duration` and `--delay` accept plain seconds or Go durations.
Passing `--frames` without `--record` writes `world.execute.ansi`.

## 9. Failure behaviour

| Missing | Result |
| --- | --- |
| `mpv` and `ffplay` | silent clock, animation still runs, player reported as `silent` |
| `ffmpeg`/`ffprobe` | no envelope; visuals fall back to a synthetic pulse derived from lyric timings; duration falls back to the last lyric plus a tail |
| any TTY | headless mode: fixed size, no raw input, frames written to stdout |
| colour | monochrome via bold/dim/reverse, layout unchanged |
| lyrics file | the boot log still runs, the stage shows the title only |

## 10. Verification

- `internal/lyric`: parse fixtures, punctuation joining, word lookup, and a
  golden check against `assets/lyrics.lrc`.
- `internal/audio`: FFT against a synthesised sine, envelope normalisation,
  and an integration pass over the real MP3 that checks the duration and that
  the chorus is louder than the intro.
- `internal/term`: screen diffing (a frame that does not change writes nothing),
  wide rune handling, truncation, colour quantisation, glitch row shifts.
- `internal/control`: keymap to action table, chrome toggles, hint timeout.
- `internal/scene`: the storyboard table maps the expected section to a set of
  timestamps, every data table entry is matched against a real lyric line, all
  sections are contiguous, and every section renders a non-empty frame. The
  frames are printed with `go test ./internal/scene -run TestRenderFrames -v`.
- Manual: `--frames 900 --record out.ansi` then `cat out.ansi` to inspect.

## 11. Progress

- [x] Asset extraction, LRC parsing with word timings and translations
- [x] Terminal layer: colour modes, diffing screen, banner font, raw mode, keys
- [x] Audio layer: clock, mpv IPC player, ffplay fallback, silent fallback
- [x] Spectrum envelope, waveform and FFT via ffmpeg
- [x] Playback controls and transport bar (section 7)
- [x] Scenes and director (section 6)
- [x] Braille canvas, geometry scenes, heart, cage, shockwave, meters
- [x] CLI wiring, README, Makefile
- [x] Tests for control, scene, terminal decoding and the mpv integration
- [ ] Human review

### Fixed after the first review

- Controls: seek and volume work, but arrow keys were decoded too eagerly, so a
  split escape sequence became a bare Escape and quit the animation. The
  decoder now follows the CSI grammar with a tolerant budget, unknown sequences
  are ignored instead of quitting, and Home/End are supported. The chosen audio
  backend is printed at startup so a silent fallback is never a mystery, and
  the transport shows mpv's own volume.
- Motion: the piece no longer leans on text alone. See section 12.
- Contrast: no background colour is painted and body text inherits the
  terminal's foreground, with `--theme light` for light terminals.

### Fixed after the second review

- The key hint row was drawn in the shadow colour, which is nearly black: it
  faded *towards* the background instead of away from it. `HintColours` now
  blends shadow to accent with the hint alpha, and `TestHintRowIsReadable`
  measures the luminance of every cell on that row in both themes.
- The screen was too empty once the panels stopped painting backgrounds, so the
  ambient layer (waveform wall, starfield, shimmer grid, beat pulse) and the
  telemetry sidecars were added. Both only paint where nothing else has, and
  the ambient keeps a one cell margin around any word.

## 12. Notes from implementation

- Text scenes sit on a blanked panel with an accent bar so the code rain never
  tramples the words. Panels deliberately paint no background colour: body text
  uses the terminal's own foreground, which is what makes the picture legible on
  dark and light themes alike. `--theme light` swaps the accent tints.
- Geometry is drawn on a braille canvas (2x4 dots per cell), which is what lets
  the verses draw the shapes they sing about: a point cloud, a self tracing
  circle, a travelling sine wave and a curve approaching a limit.
- Motion is everywhere, not only in the graphics sections: the header carries a
  live oscilloscope, the code sections carry a twelve band spectrum meter, the
  boot and shutdown logs carry a system monitor and the panic sections carry a
  scrolling memory dump. The ambient layer fills whatever is left over.
- The glitch is suppressed below a small threshold so it reads as punctuation
  rather than damage, and is disabled entirely while the title banner owns the
  screen.
- Escape sequences are decoded against the full CSI grammar with a 120 ms
  budget. Splitting them (which PTYs and SSH sessions do) and unknown sequences
  such as Home no longer collapse into a bare Escape, which used to quit.
- mpv's position reports are ignored for 400 ms after a seek, and reports that
  would drag the clock backwards are dropped, so a seek survives the stale
  property changes mpv had already queued.

## 13. Lyric-driven animation expansion (pending human review)

The lyric arc moves from construction and connection, through transformations
and abandonment, into execution and confinement. Illustrations follow those
changes rather than using the same background effect for every passage.

| Cue | Illustration |
| --- | --- |
| 00:00–00:19 — power, protection, creation, initialization, simulation | A connecting plug, shield, assembling wireframe cube, rotating world |
| 00:19–00:30 — title | Scrolling perspective floor beneath the title |
| 00:44–00:51 — AC/DC, blindness, dizziness | Flowing circuit charges; switch closes and sinusoid becomes DC at the word onset, **00:47.22**; shutters and rotating arcs |
| 00:51–00:59 — travel, A.D./B.C., unite, deeply | Reversing tunnel with two orbiting entities converging into one |
| 00:59–01:14 — stimulation, satisfaction, execution, simulation | Signal network, locking rings, radial target pulses, rotating cage |
| 01:14–01:29 — eggplant, tomato, cat, god | Large materializing illustrations; keyword particles; blinking eyes and purr ripples |
| 01:29–01:43 — gender, time, role, trance | Morphing F/M symbol, clock hands, exchanging orbiting nodes, twisting helix |
| 01:43–01:51 — vibrations, completion | Two waves approach the same phase and complete a ring/check mark |
| 01:51–02:05 — left, isolation, fragments | Connections disappear around a lone heart; the heart fragments and erases |
| 02:05–02:28 — challenge, illegal arguments | Tilting judgment scales become a rejection triangle |
| 02:42–02:57 — execution, trapped | Repeated target pulses transition into the simulation cage |
| 02:57–03:08 — algebraic expression of love | Heart traced on coordinate axes with its parametric equation |
| 03:08 onward — free / trapped | Birds depart beside the existing closing heart cage |

Implementation: `motion.go` contains the register/object/title illustrations;
`illustration.go` adds timed lower-stage illustrations to the boot, code and
decay scenes. Existing lyric logs occupy the upper stage. Compact terminals
retain the original layout when there is insufficient illustration space.
New motion uses playback time, so it freezes when paused and follows seeks.
New labels inherit terminal foreground; graphic accents use the selected theme.
The storyboard timing check accepts both line and word onsets from the LRC.

Verification: build, vet, formatting and the full Go test suite pass. Full-song
silent renders also pass at 40×18, 80×24 and 140×48 in dark and light themes;
representative frame previews were inspected. Human visual acceptance is pending.

## 13b. Animations for the quiet stretches

Found by rendering frames across every instrumental gap and listening for
what the lyric file leaves unsung.

| Time | Gap | Animation |
| --- | --- | --- |
| 0:21–0:29 | The long instrumental between "world." and "execute" | The second title line fades in as flickering static, and a link bar resolves `locating self → resolving execute → binding me → linking ;`, hitting 100% on the word onset at 0:29.25 |
| 0:29–0:44 | Verse 1 leaves the top of the plot empty | A live readout per shape: point count and spread, `r`, `C = 2πr`, `A = πr²`, `dy/dx` of the sine, then `x` and `1/x` as the limit runs to infinity |
| 2:38.97–2:41.67 | EIN DOS TROIS NE FEM LIU is one number on a bare screen | Every step closes concentric rings and rotating ticks onto the number, one lit marker per step |
| 3:28.30–3:31.9 | EXIT log fills only the top of the screen | A heart monitor: 72 bpm sinus rhythm that fails on SIGTERM at 3:29.8, decays and flatlines at 3:30.9 to `asystole`, scrolling as a real trace |

The monitor is dropped below 22 rows and stays clear of the telemetry strip;
the verse readout is dropped below 64 columns.

## 13e. Animation timing

Measured against real spectral-flux onsets, the lyric timestamps the animation
is keyed to are accurate: 42 shouted keywords sit at a median -5 ms with the
middle half inside -15..+5 ms. The remaining lag was in the display path, so the
fixes are all on the animation side and the audio is untouched:

- Each frame is rendered for the moment it will be shown, one frame period
  after the clock reading, instead of the moment it was drawn. Frame dumps and a
  paused clock are not shifted.
- The waveform wall rises with a 10 ms time constant and falls with a 60 ms
  one, so beats land on time and the wall still glides down (it used a
  symmetric 39 ms).
- Analysis windows are centred on their frame time (section 13a).
- `[` `]` and `{` `}` fine-tune the animation against the audio for a
  particular terminal or display, the readout `sync +20ms` shows in the
  transport, and the value is saved to the user config directory and used by
  the next run unless `--delay` is given. Only the animation clock moves; mpv is
  never touched.

## 13d. Visibility and lyric fixes

- Rain density compared a column's index to a fraction of the width, so quiet
  sections rained only on the left edge. Every column now has a golden-ratio
  lane and is lit when its lane is under the density, so any density spreads
  evenly across the width.
- Text painted in the palette's `Shadow` colour is invisible, because that is
  the background. The telemetry graph and hex dump, the spectrum meter and the
  header scope now use `Palette.Faint`, and fades that reached zero were given
  floors. `term.Mix` no longer turns `ColorDefault` into black, which had made
  the EXECUTION hit fade to black blocks; it fades to `Palette.Ink` instead.
  `TestNoTextVanishesIntoTheBackground` scans the whole song in both themes.
- The trap scene stopped drawing the lyric a second before "OVE" was sung. It
  holds until the countdown and keeps "Trapped in" above "LO-O-OVE".
- Every shouted lyric word of four letters or more sends a shockwave ring
  across the empty space, and the love sections rain hearts.

## 13c. Simulation pass

The whole song was rendered every 250 ms with only the scene layers, counting
painted cells, missing lyrics and clipped text. The emptiest and the sloppiest
moments got work:

| Time | Finding | Change |
| --- | --- | --- |
| 0:00–0:00.9 | The screen just appeared | CRT power-on: a bright line opens sideways, then up into the boot log |
| 1:50.94–1:58.38 | Abandonment thinned to ~145 cells | A ping log (`ping you seq=NN timeout`) and a packet loss gauge climbing to 100% |
| 1:58.38–2:05.67 | Fragment phrases overprinted each other and ran off the edge (21 phrases in 9 rows) | One phrase per row, a rotating subset every 2.4 s, drifting across in alternating directions with brief tears and growing character corruption, clipped before the telemetry strip |
| 3:25.86–3:28.30 | 1.6 s of a single dim word after the last hit | CRT power-off: the picture collapses to a line, the line shrinks to a dot, the dot fades, then the exit log |

## 13a. Performance pass

Measured with `go test ./internal/scene -bench Frame` (160×45, truecolor):
0.305 ms → 0.166 ms per frame, 1760 → 127 allocations, 43 KB → 9 KB.

- `term`: ASCII fast path for `RuneWidth`; SGR built with `append` into a reused
  buffer and emitted as a delta when only colours change; each changed frame is
  wrapped in synchronized output (`?2026`) so terminals that support it never
  tear.
- `scene`: a reseedable `FrameRand` replaces a fresh `rand.Source` per frame and
  holds its seed for 48 ms, so glitch and shake are frame-rate independent; the
  section lookup is a binary search; the ambient wall smooths with a
  time-constant instead of a per-frame factor.
- `audio`: analysis frames run on all cores with cached FFT twiddles (0.85 s →
  0.27 s); energy and bands are interpolated between frames like the waveform,
  and every analysis window is centred on its frame time rather than starting
  there, which removed a fixed ~46 ms lead on every beat reaction.
- `main`: a key press no longer forces an extra frame at 30 fps and above; the
  next tick draws it.

## 14. Out of scope

- Video files, GIFs, or browser output. The output is a terminal.
- Recording the audio mix; `--record` captures frames only.
- Windows terminal handling; the raw mode path targets POSIX.
