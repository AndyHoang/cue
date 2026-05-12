package player

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/SuperCoolPencil/cue/internal/domain"
)

// Launcher launches media URLs in an external player
type Launcher struct {
	command  string   // configured player command, empty for system default
	args     []string // additional arguments for the player
	seekFlag string   // user-configured seek flag (e.g., "--start=%d"), overrides table lookup
	logger   *slog.Logger
}

// PlayerDef defines a player binary and its seek flag format
type PlayerDef struct {
	Binary   string
	SeekFlag string // Use %d for seconds placeholder, e.g., "--start=%d" or "-ss %d"
}

// Platform-specific player lists, ordered by priority (first match wins)
var linuxPlayers = []PlayerDef{
	{Binary: "mpv", SeekFlag: "--start=%d"},
	{Binary: "vlc", SeekFlag: "--start-time=%d"},
	{Binary: "celluloid", SeekFlag: "--mpv-start=%d"},
	{Binary: "haruna", SeekFlag: "--start=%d"},
	{Binary: "smplayer", SeekFlag: "-ss %d"},
	{Binary: "mplayer", SeekFlag: "-ss %d"},
}

var darwinPlayers = []PlayerDef{
	{Binary: "iina", SeekFlag: "--mpv-start=%d"},
	{Binary: "mpv", SeekFlag: "--start=%d"},
	{Binary: "vlc", SeekFlag: "--start-time=%d"},
}

// NewLauncher creates a new Launcher
// seekFlag is optional - if empty, we look up the flag from our known players table
func NewLauncher(command string, args []string, seekFlag string, logger *slog.Logger) *Launcher {
	if logger == nil {
		logger = slog.Default()
	}

	return &Launcher{
		command:  command,
		args:     args,
		seekFlag: seekFlag,
		logger:   logger,
	}
}

// Launch opens a media URL in the configured player or auto-detected player.
// Any external subtitle tracks in `media` are side-loaded into mpv-family players.
func (l *Launcher) Launch(media domain.PlayableMedia, startOffset time.Duration) (*exec.Cmd, string, error) {
	offsetSecs := int(startOffset.Seconds())

	// Tier 1: User configured a specific player
	if l.command != "" {
		l.logger.Info("using configured player", "command", l.command)
		return l.launchConfigured(media, offsetSecs)
	}

	// Tier 2: Auto-detect known players
	if player, found := l.detectPlayer(); found {
		l.logger.Info("auto-detected player", "binary", player.Binary)
		return l.execPlayer(player, media, offsetSecs)
	}

	// Tier 3: System default fallback (xdg-open/open)
	l.logger.Warn("no video players found, falling back to system default")
	if offsetSecs > 0 {
		l.logger.Warn("resume not supported with system default player - starting from beginning")
	}
	if len(media.Subtitles) > 0 {
		l.logger.Warn("external subtitles not supported with system default player - some tracks may be missing")
	}
	cmd, err := l.launchDefault(media.URL)
	return cmd, "", err
}

// subFileArgs returns the player-specific args needed to side-load each external
// subtitle. Returns nil when the binary has no known sub-file flag.
func subFileArgs(binary string, subs []domain.Subtitle) []string {
	if len(subs) == 0 {
		return nil
	}
	bin := strings.ToLower(filepath.Base(binary))
	switch bin {
	case "mpv", "iina", "celluloid", "haruna":
		// mpv, IINA and other mpv-frontends accept multiple --sub-file flags.
		// IINA's CLI is `iina-cli`, but mpv-passthrough flags also work via
		// the `--mpv-` prefix used in seek flags; --sub-file works directly
		// for mpv/celluloid/haruna. IINA accepts `--mpv-sub-file=` too.
		prefix := "--sub-file="
		if bin == "iina" {
			prefix = "--mpv-sub-file="
		}
		args := make([]string, 0, len(subs))
		for _, s := range subs {
			if s.URL == "" {
				continue
			}
			args = append(args, prefix+s.URL)
		}
		return args
	case "vlc":
		// VLC supports only a single :sub-file. If the user has multiple,
		// pick the default (or first) so they at least get one.
		pick := subs[0]
		for _, s := range subs {
			if s.Default {
				pick = s
				break
			}
		}
		if pick.URL == "" {
			return nil
		}
		return []string{":sub-file=" + pick.URL}
	default:
		return nil
	}
}

// detectPlayer returns the first available player from the platform-specific list
func (l *Launcher) detectPlayer() (PlayerDef, bool) {
	var candidates []PlayerDef

	switch runtime.GOOS {
	case "darwin":
		candidates = darwinPlayers
	case "linux":
		candidates = linuxPlayers
	default:
		return PlayerDef{}, false
	}

	for _, p := range candidates {
		if path, err := exec.LookPath(p.Binary); err == nil && path != "" {
			return p, true
		}
	}
	return PlayerDef{}, false
}

// execPlayer launches the detected player with optional seek offset
func (l *Launcher) execPlayer(player PlayerDef, media domain.PlayableMedia, offsetSecs int) (*exec.Cmd, string, error) {
	args := []string{}
	var ipcSocket string

	// Enable IPC for mpv
	if player.Binary == "mpv" {
		ipcSocket = filepath.Join(os.TempDir(), fmt.Sprintf("cue-mpv-%d.sock", time.Now().UnixNano()))
		args = append(args, "--input-ipc-server="+ipcSocket)
	}

	// Add seek flag if we have an offset and the player supports it
	if offsetSecs > 0 && player.SeekFlag != "" {
		formattedFlag := fmt.Sprintf(player.SeekFlag, offsetSecs)
		// Split flags like "-ss 10" into separate args
		args = append(args, strings.Fields(formattedFlag)...)
	}

	if subArgs := subFileArgs(player.Binary, media.Subtitles); len(subArgs) > 0 {
		args = append(args, subArgs...)
	} else if len(media.Subtitles) > 0 {
		l.logger.Warn("external subtitles not supported by player - skipping",
			"binary", player.Binary, "count", len(media.Subtitles))
	}

	args = append(args, media.URL)

	l.logger.Debug("executing player", "binary", player.Binary, "args", args)
	cmd := exec.Command(player.Binary, args...)
	if err := cmd.Start(); err != nil {
		return nil, "", err
	}
	return cmd, ipcSocket, nil
}

// launchConfigured launches the media using the user-configured player
func (l *Launcher) launchConfigured(media domain.PlayableMedia, offsetSecs int) (*exec.Cmd, string, error) {
	args := append([]string{}, l.args...)

	// Add seek offset: user-configured flag takes precedence, then table lookup
	if offsetSecs > 0 {
		seekFlag := l.seekFlag
		if seekFlag == "" {
			// Fall back to table lookup for known players
			seekFlag = l.lookupSeekFlag(l.command)
		}

		if seekFlag != "" {
			formattedFlag := fmt.Sprintf(seekFlag, offsetSecs)
			args = append(args, strings.Fields(formattedFlag)...)
		} else {
			l.logger.Warn("cannot set start offset - unknown player, configure start_flag in config",
				"command", l.command, "offset", offsetSecs)
		}
	}

	if subArgs := subFileArgs(l.command, media.Subtitles); len(subArgs) > 0 {
		args = append(args, subArgs...)
	} else if len(media.Subtitles) > 0 {
		l.logger.Warn("external subtitles not supported by configured player - skipping",
			"command", l.command, "count", len(media.Subtitles))
	}

	args = append(args, media.URL)

	l.logger.Debug("launching configured player", "command", l.command, "args", args)

	// On macOS, try 'open -a' if command not in PATH (for GUI apps)
	if runtime.GOOS == "darwin" {
		if _, err := exec.LookPath(l.command); err != nil {
			cmd, err := l.launchMacOSApp(l.command, args)
			return cmd, "", err
		}
	}

	// For manual config, we check if it's mpv to enable IPC
	var ipcSocket string
	if l.command == "mpv" || strings.HasSuffix(l.command, "/mpv") {
		ipcSocket = filepath.Join(os.TempDir(), fmt.Sprintf("cue-mpv-%d.sock", time.Now().UnixNano()))
		args = append([]string{"--input-ipc-server=" + ipcSocket}, args...)
	}

	cmd := exec.Command(l.command, args...)
	if err := cmd.Start(); err != nil {
		return nil, "", err
	}
	return cmd, ipcSocket, nil
}

// lookupSeekFlag finds the seek flag for a known player binary
func (l *Launcher) lookupSeekFlag(binary string) string {
	for _, p := range linuxPlayers {
		if p.Binary == binary {
			return p.SeekFlag
		}
	}
	for _, p := range darwinPlayers {
		if p.Binary == binary {
			return p.SeekFlag
		}
	}
	return ""
}

// launchMacOSApp launches a macOS GUI app using 'open -a'
func (l *Launcher) launchMacOSApp(appName string, playerArgs []string) (*exec.Cmd, error) {
	cmdArgs := []string{"-a", appName}
	if len(playerArgs) > 0 {
		cmdArgs = append(cmdArgs, "--args")
		cmdArgs = append(cmdArgs, playerArgs...)
	}

	l.logger.Debug("using macOS 'open -a'", "app", appName, "args", cmdArgs)
	cmd := exec.Command("open", cmdArgs...)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

// launchDefault opens the URL using the system default handler
func (l *Launcher) launchDefault(url string) (*exec.Cmd, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		// Linux and other Unix-like systems
		cmd = exec.Command("xdg-open", url)
	}

	l.logger.Debug("launching with system default", "os", runtime.GOOS, "url", url)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

// Service orchestrates playback operations
type Service struct {
	launcher  *Launcher
	playback  domain.PlaybackClient
	scrobbler *Scrobbler
	logger    *slog.Logger
}

// NewService creates a new playback service
func NewService(launcher *Launcher, playback domain.PlaybackClient, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		launcher:  launcher,
		playback:  playback,
		scrobbler: NewScrobbler(playback, logger),
		logger:    logger,
	}
}

// Play starts playback of a media item from the beginning
func (s *Service) Play(ctx context.Context, item domain.MediaItem) (PlaybackHandle, error) {
	return s.playItem(ctx, item, 0)
}

// Resume starts playback from the saved position
func (s *Service) Resume(ctx context.Context, item domain.MediaItem) (PlaybackHandle, error) {
	return s.playItem(ctx, item, item.ViewOffset)
}

// playItem resolves URL and launches player
func (s *Service) playItem(ctx context.Context, item domain.MediaItem, offset time.Duration) (PlaybackHandle, error) {
	media, err := s.playback.ResolvePlayable(ctx, item.ID)
	if err != nil {
		s.logger.Error("failed to resolve playable URL", "error", err, "itemID", item.ID)
		return PlaybackHandle{}, err
	}

	s.logger.Info("launching playback",
		"title", item.Title, "itemID", item.ID, "offset", offset, "subtitles", len(media.Subtitles))

	cmd, ipcSocket, err := s.launcher.Launch(media, offset)
	if err != nil {
		return PlaybackHandle{}, err
	}

	// Start monitoring progress
	return s.scrobbler.Monitor(ctx, cmd, ipcSocket, item), nil
}

// MarkWatched marks an item as fully watched
func (s *Service) MarkWatched(ctx context.Context, itemID string) error {
	return s.playback.MarkPlayed(ctx, itemID)
}

// MarkUnwatched marks an item as unwatched
func (s *Service) MarkUnwatched(ctx context.Context, itemID string) error {
	return s.playback.MarkUnplayed(ctx, itemID)
}
