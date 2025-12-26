package domain

import (
	"errors"

	"github.com/google/uuid"
)

type Playlist struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Platform    Platform `json:"platform"`
	PlatformID  string   `json:"platformId"`
	OwnerID     string   `json:"ownerId,omitempty"`
	ImageURL    string   `json:"imageUrl,omitempty"`
	TrackCount  int      `json:"trackCount"`
	Tracks      []*Track `json:"tracks,omitempty"`
}

func NewPlaylist(name string, platform Platform, platformID string) (*Playlist, error) {
	if name == "" {
		return nil, errors.New("playlist name cannot be empty")
	}
	if !platform.IsValid() {
		return nil, errors.New("invalid platform")
	}
	if platformID == "" {
		return nil, errors.New("platform ID cannot be empty")
	}

	return &Playlist{
		ID:         uuid.New().String(),
		Name:       name,
		Platform:   platform,
		PlatformID: platformID,
		Tracks:     make([]*Track, 0),
	}, nil
}

func (p *Playlist) AddTrack(track *Track) {
	if track != nil {
		p.Tracks = append(p.Tracks, track)
		p.TrackCount = len(p.Tracks)
	}
}

func (p *Playlist) AddTracks(tracks []*Track) {
	for _, track := range tracks {
		p.AddTrack(track)
	}
}

func (p *Playlist) WithDescription(description string) *Playlist {
	p.Description = description
	return p
}

func (p *Playlist) WithOwner(ownerID string) *Playlist {
	p.OwnerID = ownerID
	return p
}

func (p *Playlist) WithImage(imageURL string) *Playlist {
	p.ImageURL = imageURL
	return p
}

func (p *Playlist) GetTrackIDs() []string {
	ids := make([]string, len(p.Tracks))
	for i, track := range p.Tracks {
		ids[i] = track.PlatformID
	}
	return ids
}
