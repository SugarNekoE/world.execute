// Package assets embeds the track and its lyrics so the binary runs without
// the assets directory next to it.
package assets

import "embed"

// Files holds world-execute-me.mp3 and lyrics.lrc.
//
//go:embed world-execute-me.mp3 lyrics.lrc
var Files embed.FS
