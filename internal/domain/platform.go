package domain

type Platform string

const (
	PlatformSpotify Platform = "SPOTIFY"
	PlatformYouTube Platform = "YOUTUBE"
)

func (p Platform) IsValid() bool {
	switch p {
	case PlatformSpotify, PlatformYouTube:
		return true
	default:
		return false
	}
}

func (p Platform) String() string {
	return string(p)
}

func ParsePlatform(s string) (Platform, bool) {
	p := Platform(s)
	return p, p.IsValid()
}
