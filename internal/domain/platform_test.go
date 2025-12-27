package domain

import "testing"

func TestPlatform_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		want     bool
	}{
		{"valid spotify", PlatformSpotify, true},
		{"valid youtube", PlatformYouTube, true},
		{"empty string", Platform(""), false},
		{"invalid platform", Platform("TIDAL"), false},
		{"lowercase", Platform("spotify"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.platform.IsValid(); got != tt.want {
				t.Errorf("Platform.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlatform_String(t *testing.T) {
	tests := []struct {
		platform Platform
		want     string
	}{
		{PlatformSpotify, "SPOTIFY"},
		{PlatformYouTube, "YOUTUBE"},
	}

	for _, tt := range tests {
		t.Run(string(tt.platform), func(t *testing.T) {
			if got := tt.platform.String(); got != tt.want {
				t.Errorf("Platform.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParsePlatform(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Platform
		valid bool
	}{
		{"parse spotify", "SPOTIFY", PlatformSpotify, true},
		{"parse youtube", "YOUTUBE", PlatformYouTube, true},
		{"parse invalid", "INVALID", Platform("INVALID"), false},
		{"parse empty", "", Platform(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, valid := ParsePlatform(tt.input)
			if got != tt.want || valid != tt.valid {
				t.Errorf("ParsePlatform(%q) = (%v, %v), want (%v, %v)",
					tt.input, got, valid, tt.want, tt.valid)
			}
		})
	}
}
