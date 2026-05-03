package domain

import (
	"testing"
	"time"
)

func TestMediaItemFormattingAndStatus(t *testing.T) {
	item := MediaItem{Duration: 95 * time.Minute, Height: 2160, FileSize: 3 * 1024 * 1024 * 1024, AudioChannels: 6}
	if item.FormattedDuration() != "1h 35m" {
		t.Fatalf("duration = %q", item.FormattedDuration())
	}
	if item.Resolution() != "4K" {
		t.Fatalf("resolution = %q", item.Resolution())
	}
	if item.FormattedFileSize() != "3.0 GB" {
		t.Fatalf("file size = %q", item.FormattedFileSize())
	}
	if item.ChannelLayout() != "5.1" {
		t.Fatalf("channel layout = %q", item.ChannelLayout())
	}
	item.ViewOffset = time.Minute
	if item.WatchStatus() != WatchStatusInProgress || !item.ShouldResume() {
		t.Fatalf("watch status = %v resume=%v", item.WatchStatus(), item.ShouldResume())
	}
	item.IsPlayed = true
	if item.WatchStatus() != WatchStatusWatched || item.ShouldResume() {
		t.Fatalf("watch status = %v resume=%v", item.WatchStatus(), item.ShouldResume())
	}
}

func TestShowSeasonPlaylistDescriptions(t *testing.T) {
	show := Show{SeasonCount: 2, EpisodeCount: 10, UnwatchedCount: 5}
	if show.GetDescription() != "2 Seasons" || show.WatchStatus() != WatchStatusInProgress {
		t.Fatalf("show desc=%q status=%v", show.GetDescription(), show.WatchStatus())
	}
	season := Season{SeasonNum: 0, EpisodeCount: 1}
	if season.DisplayTitle() != "Specials" || season.GetDescription() != "1 Episode" {
		t.Fatalf("season title=%q desc=%q", season.DisplayTitle(), season.GetDescription())
	}
	playlist := Playlist{ItemCount: 2}
	if playlist.GetDescription() != "2 items" {
		t.Fatalf("playlist desc=%q", playlist.GetDescription())
	}
}
