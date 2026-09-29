# world.execute(me);

A synchronised, animated lyric video for Mili's *world.execute (me) ;* that
plays **inside the terminal**. The picture is made of coloured text, the timing
comes from the word level lyrics embedded in the MP3, and the audio is the
bundled track.

```
                                 WORLD
                      █   █  ███  ████  █     ████
                      █   █ █   █ █   █ █     █   █
                      █   █ █   █ ████  █     █   █
                      █ █ █ █   █ █ █   █     █   █
                      ██ ██ █   █ █  █  █     █   █  ██
                      █   █  ███  █   █ █████ ████   ██

                        Mili — Miracle Milk (2015)
```

The song is written from the point of view of an artificial intelligence asking
its creator for a purpose. The program answers with the machine's own paper
trail: a boot log, pseudocode love, a panic, and a final `EXECUTION`.

## Requirements

- Go 1.26 or newer to build.
- **mpv** (preferred) or **ffplay** to hear the track. Without one the
  animation still runs, silently.
- **ffmpeg** and **ffprobe** to make the visuals react to the music. Without
  them the animation falls back to a plain pulse.
- A terminal with truecolor for the best result. 256 colour, 16 colour and
  monochrome are supported, as is a pipe or a file instead of a terminal.

## Build and run

```sh
make run                 # build and play
make build               # ./bin/world.execute
./bin/world.execute --help
```

The binary carries the track and its lyrics, so `bin/world.execute` runs from
any directory on its own. An `assets/` directory found next to the working
directory or the binary wins, which is how you swap the track; otherwise the
embedded copies are written once to your cache directory and played from there.

Then:

```sh
./bin/world.execute                  # play the whole song
./bin/world.execute --start 2:27     # jump straight to the executions
./bin/world.execute --player silent  # no audio, animation only
./bin/world.execute --color 16       # for a plain terminal
./bin/world.execute --frames 900 --record out.ansi --start 1:00
cat out.ansi                         # replay a frame dump
```

## What you see

The song is drawn as the machine's own telemetry, and the whole picture is
driven by the audio:

- a **boot log** whose lines arrive with the sung words;
- a connecting **power plug**, protective shield, assembling cube and rotating
  world during initialization;
- the title **assembling itself** above a moving perspective grid;
- the first verse **drawing the geometry it sings about** — a set of points, a
  circle tracing its own circumference, a travelling sine wave, a curve that
  approaches a limit it never reaches — on a braille plot;
- an **AC/DC circuit** with flowing charges, a moving switch and a sine wave
  becoming a steady line on the sung “DC”; shutters and spinning arcs follow;
- a reversing **time tunnel**, two entities merging, a morphing F/M symbol,
  spinning clock, swapping roles and a trance helix;
- large **materializing objects**: eggplant, tomato, blinking cat with purr
  ripples, and a radiant heart, with particles on their keyword cues;
- **signal networks**, locking targets and rotating simulation cages in the
  choruses; two waves synchronize on **COMPLETION**;
- connections disappearing into **isolation**, a heart breaking into fragments,
  and judgment scales becoming an error symbol;
- a **panic and a stack trace** for the illegal arguments;
- **EXECUTION**, twelve times, as an expanding shockwave with the word punched
  through it, then a countdown;
- a **heart** drawn parametrically for LO-O-OVE, a **cage closing** around it
  while the last chorus counts down to the verdict, and a shutdown log at the
  end;
- the **algebraic expression of love** tracing a heart on coordinate axes,
  then birds escaping beside the trapped heart;
- and, filling every gap the scenes leave: a **waveform wall** along the bottom,
  a slow **starfield**, a shimmer grid, a **pulse on the beat**, a live
  **oscilloscope** in the header, a **spectrum meter** in the code sections and
  a **system monitor** or **memory dump** beside the boot and panic logs — so
  nothing on screen is ever static.

The main objects are drawn as **dense ASCII art**. Each one is a shape filled
with streaming code: every column falls at its own speed through hex, binary and
symbols, the outline is drawn in heavy glyphs, and a scanner band sweeps across.
A tomato with a calyx, an eggplant, a tabby cat with `@` eyes and whiskers, a sun
with a `<3` heart, a rotating character-map Earth, a shield with a `#` check
mark, a warning triangle, a hex-filled memory block and a heart made of `0 1 < 3`
all get the same treatment, wrapped in the furniture of a targeting system:
corner brackets that breathe, a crosshair with tick marks, a hex dump scrolling
up the side, live readouts and a progress bar.

On top of the scenes, a layer of cinematic effects, all driven by the music and
the lyrics and all careful never to touch the text:

- **data streams**: rows of hex, addresses and packet words (`0xDEADBEEF`,
  `SYN`, `192.168.0.13:443`) scroll through the empty space, alternating
  direction, with a bright band travelling along each;
- **shockwave rings and falling sparks** that leave the centre every time a
  shouted word such as DIMENSION or EXECUTION is sung;
- a **chromatic split** on the big hits, where the picture separates into a red
  and a cyan ghost for a moment;
- every sung word **flashes** as it lands, and each section's name **decodes**
  out of noise in the header;
- a soft **vignette** towards the edges, a **wipe** between sections, and a CRT
  **power-on** at the start and **power-off** at the final execution.

## Controls

The transport bar sits at the bottom of the screen and the key hints fade away
after five seconds; press any key to bring them back.

| Key | Action |
| --- | --- |
| `space`, `p` | pause or resume |
| `→`, `l` / `←`, `h` | seek 5 s forward / back |
| `.` / `,` | seek 1 s forward / back |
| `1`–`9` | jump to a chapter (boot, title, object creation, stimulations, switch gender, abandonment, illegal arguments, execution, love) |
| `0`, `r` | restart |
| `End` | jump to the last second |
| `↑`, `+` / `↓`, `-` | volume up / down |
| `m` | mute |
| `[` `]` | shift the animation 10 ms later / earlier against the audio |
| `{` `}` | the same in 50 ms steps |
| `s` | toggle the info rows |
| `?` | help |
| `q`, `Esc`, `Ctrl-C` | quit |

Seeking and volume go to mpv over its JSON IPC, so the audio follows. The
transport row shows mpv's own volume, so you can see the change land. With
ffplay, seeking restarts the process at the new offset and volume is not
adjustable. With the silent backend only the animation moves — the program says
which backend it picked when it starts.

## Flags

| Flag | Default | Meaning |
| --- | --- | --- |
| `--audio` | `assets/world-execute-me.mp3` | track to play |
| `--lyrics` | `assets/lyrics.lrc` | lyric file |
| `--player` | `auto` | `auto`, `mpv`, `ffplay` or `silent` |
| `--lang` | `en` | `en`, `zh` or `both`; only `en` has word level timings, the others reveal whole lines |
| `--fps` | `60` | animation frame rate |
| `--color` | `auto` | `auto`, `truecolor`, `256`, `16` or `none` |
| `--theme` | `dark` | `dark` or `light`; text inherits your terminal's own foreground |
| `--start` | `0` | start offset |
| `--duration` | whole track | stop after this much playback |
| `--charset` | `ascii` | `ascii` writes plain ASCII only, so blocks, braille and box lines become `# - | / \ .`; `unicode` keeps them |
| `--delay` | saved value, else `0` | shifts the animation against the audio (positive earlier, negative later); `[` `]` adjust it live and it is remembered |
| `--volume` | `80` | initial volume |
| `--seed` | clock | seed for the code rain |
| `--no-analysis` | off | skip the ffmpeg spectrum pass |
| `--width`, `--height` | terminal | force the cell grid |
| `--frames` | `0` | render this many frames as fast as possible |
| `--record` | — | write frames to a file instead of the terminal |
| `--quiet`, `--verbose` | off | quieter or chattier |

`--start`, `--duration` and `--delay` accept either seconds (`--start 147.5`)
or Go durations (`--start 2m27s`).

## How it works

```
cmd/world.execute      flag parsing, wiring and the frame loop
internal/config        the resolved run configuration
internal/lyric         LRC parsing, word timings, timeline lookup
internal/audio         player process, clock, mpv IPC, FFT envelope
internal/term          colour, diffing screen buffer, banner font, raw mode
internal/control       keymap and playback actions
internal/scene         the animation: storyboard, layers, transport bar
```

- **Lyrics.** `assets/lyrics.lrc` was extracted from the MP3's `lyrics-XXX`
  tag. It is enhanced LRC: `[line time]<word time>word …` for the English
  lines, with the Chinese translation on a second line sharing the same
  timestamp. The parser pairs them and keeps the word timings, which is what
  makes the words light up exactly when they are sung.
- **Sync.** The animation clock is a monotonic timer, corrected continuously by
  mpv's reported playback position, so a slow frame never drifts.
- **Envelope.** ffmpeg decodes the track to mono PCM once at startup; a Hann
  window and an FFT per frame produce twelve log spaced bands, an RMS envelope
  and a downsampled waveform, which drive the oscilloscope, the shapes, the
  meters, the code rain and the glitches.
- **Drawing.** Curves are plotted on a braille canvas (two by four dots per
  cell), which gives smooth geometry from plain characters; text scenes sit on
  a blanked panel so the rain never tramples the words.
- **Rendering.** Scenes paint into a cell buffer; only the cells that changed
  since the previous frame are written, with a cursor move for each run. At
  60 fps a quiet frame costs a few hundred bytes.
- **Readability.** Body text uses the terminal's own foreground and no
  background is painted, so the picture stays legible on any terminal theme.
- **Determinism.** Every scene is a pure function of the playback position and
  the seed, so `--start` and `--frames` reproduce the same picture every time.

The storyboard, the exact timings and the design decisions are in
[SPEC.md](SPEC.md).

## Troubleshooting

- **No sound.** Install mpv or ffplay, or pass `--player silent` on purpose.
  The chosen backend is printed at startup; `--verbose` also shows the player's
  own errors.
- **Seek or volume does nothing.** They need mpv; check the backend printed at
  startup. With `--player silent` there is no audio to move.
- **The picture is too dark or too bright.** `--theme light` is for terminals
  with a light background. `--color 256` helps when truecolor is reported but
  not honoured.
- **Audio and picture are slightly apart.** Press `[` or `]` while it plays to
  move the animation 10 ms later or earlier (`{` `}` for 50 ms). Only the
  animation moves, never the audio, and the value is remembered for the next
  run. `--delay -0.05s` or `--delay 0.05s` sets it from the command line.
- **It is too busy.** `--no-analysis` stops the visuals reacting to the music,
  and the `s` key hides the transport rows.

## Assets and licensing

The lyrics and the recording belong to Mili and their label and are included
here, and embedded in the binary, only so the program can play the song it was
written for. No licence is granted for the audio or the lyrics.

The Go code is a separate work, licensed under the GNU General Public License,
version 3 only. See `LICENSE`.
