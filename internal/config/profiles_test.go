package config

import "testing"

func TestApplyCurrentProfile(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CurrentProfile = "home"
	cfg.Profiles["home"] = ProfileConfig{Server: ServerConfig{
		Type:  SourceTypePlex,
		URL:   "http://plex",
		Token: "token",
	}}

	cfg.applyCurrentProfile()

	if cfg.Server.URL != "http://plex" || cfg.Server.Token != "token" {
		t.Fatalf("server not applied from profile: %#v", cfg.Server)
	}
}

func TestRememberCurrentProfile(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CurrentProfile = "jellyfin"
	cfg.Server = ServerConfig{
		Type:  SourceTypeJellyfin,
		URL:   "http://jellyfin",
		Token: "token",
	}

	cfg.rememberCurrentProfile()

	profile, ok := cfg.Profiles["jellyfin"]
	if !ok {
		t.Fatal("profile was not stored")
	}
	if profile.Server.URL != "http://jellyfin" {
		t.Fatalf("profile = %#v", profile)
	}
}
