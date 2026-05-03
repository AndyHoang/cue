# Cue

![Cue Demo](demo.gif?v=3)

> A fast terminal client for browsing and playing media from Plex and Jellyfin servers

## Features

-  **Automatic Scrobbling**: Real-time playback progress sync with Plex and Jellyfin.
-  **Auto-Mark Watched**: Items are automatically marked as watched on the server when reaching 90% completion.
-  **Fuzzy search** across your entire library.
-  **Keyboard-first interface** with Vim-style navigation.
-  **Playlist management** and queueing.
-  **Watch status tracking** and smart resume with visual feedback.
-  **Inspector panel** for detailed metadata and progress bars.
-  **Fast, cached browsing** with progressive loading.

## Quick Start

### Installation

**Download** from [Releases](https://github.com/SuperCoolPencil/cue/releases) or install with Go:

```bash
go install github.com/SuperCoolPencil/cue/cmd/cue@latest
```

### First Run

Launch Cue and follow the interactive setup:

```bash
cue
```

You'll be prompted to enter your server URL. Cue automatically detects whether it's a Plex or Jellyfin server and guides you through the appropriate authentication.

## Usage

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `↑` `↓` `j` `k` | Navigate up/down |
| `←` `→` `h` `l` | Navigate left/right (columns) |
| `Enter` | Play/Resume item |
| `p` | Play from start |
| `w` / `u` | Mark watched / unwatched |
| `f` | Global search |
| `/` | Local filter (current column) |
| `Space` | Manage playlists |
| `a` | Add/remove from queue |
| `N` | Play next episode |
| `s` | Sort options |
| `i` | Toggle inspector panel |
| `r` / `R` | Refresh library / all |
| `g` / `G` | Jump to top / bottom |
| `Ctrl+u` / `d` | Page up / half-page down |
| `L` | Logout |
| `?` | Show help |
| `q` | Quit/Back |

## Configuration

Config file: `~/.config/cue/config.yaml` (created on first run).

### Playback Scrobbling
Cue uses **mpv's JSON-RPC IPC** to track real-time progress. For the best experience, ensure `mpv` is installed. When using `mpv`, Cue will:
- Save your position every 10 seconds.
- Show "Saved MM:SS to server" in the status bar.
- Automatically mark the item as watched on your server once you reach 90% of the duration.

Other players (VLC, IINA, etc.) are supported for playback, but may only support "mark watched" on process exit.

## Attribution

Cue is forked from [Kino](https://github.com/mmcdole/kino), originally created by Matthew McDole. The original MIT license notice is preserved in `LICENSE`.

## License

MIT
