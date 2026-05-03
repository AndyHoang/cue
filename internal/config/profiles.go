package config

import "sort"

// ProfileConfig stores a named server profile.
type ProfileConfig struct {
	Server ServerConfig `mapstructure:"server"`
}

// ProfileNames returns configured profile names in stable insertion-free order.
func (c *Config) ProfileNames() []string {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ApplyProfileForUI switches the active server profile in memory.
func (c *Config) ApplyProfileForUI(name string) {
	if c == nil {
		return
	}
	c.CurrentProfile = name
	c.applyCurrentProfile()
}
