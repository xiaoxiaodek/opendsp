package vast

import (
	"encoding/xml"
	"fmt"
	"strings"
)

type VAST struct {
	XMLName xml.Name `xml:"VAST"`
	Version string   `xml:"version,attr"`
	Ads     []Ad     `xml:"Ad"`
}

type Ad struct {
	ID     string `xml:"id,attr"`
	InLine InLine `xml:"InLine"`
}

type InLine struct {
	AdSystem   string    `xml:"AdSystem"`
	AdTitle    string    `xml:"AdTitle"`
	Impression []CDATA   `xml:"Impression"`
	Creatives  Creatives `xml:"Creatives"`
	Icon       *Icon     `xml:"Icon,omitempty"`
}

type Creatives struct {
	Creative []Creative `xml:"Creative"`
}

type Creative struct {
	ID     string `xml:"id,attr,omitempty"`
	Linear Linear `xml:"Linear"`
}

type Linear struct {
	Duration       string          `xml:"Duration"`
	TrackingEvents *TrackingEvents `xml:"TrackingEvents,omitempty"`
	VideoClicks    *VideoClicks    `xml:"VideoClicks,omitempty"`
	MediaFiles     MediaFiles      `xml:"MediaFiles"`
}

type TrackingEvents struct {
	Trackings []Tracking `xml:"Tracking"`
}

type Tracking struct {
	Event string `xml:"event,attr"`
	CDATA CDATA  `xml:",innerxml"`
}

type VideoClicks struct {
	ClickThrough   *ClickThrough   `xml:"ClickThrough,omitempty"`
	ClickTrackings []ClickTracking `xml:"ClickTracking,omitempty"`
}

type ClickThrough struct {
	Type  string `xml:"type,attr,omitempty"`
	CDATA CDATA  `xml:",innerxml"`
}

type ClickTracking struct {
	CDATA CDATA `xml:",innerxml"`
}

type MediaFiles struct {
	MediaFile []MediaFile `xml:"MediaFile"`
}

type MediaFile struct {
	Delivery string `xml:"delivery,attr"`
	Type     string `xml:"type,attr"`
	Width    int32  `xml:"width,attr"`
	Height   int32  `xml:"height,attr"`
	URL      CDATA  `xml:",innerxml"`
}

type Icon struct {
	Program        string         `xml:"program,attr"`
	Width          int32          `xml:"width,attr"`
	Height         int32          `xml:"height,attr"`
	StaticResource StaticResource `xml:"StaticResource"`
}

type StaticResource struct {
	CreativeType string `xml:"creativeType,attr"`
	CDATA        CDATA  `xml:",innerxml"`
}

type CDATA string

func (c CDATA) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(struct {
		Data string `xml:",cdata"`
	}{string(c)}, start)
}

type AdParams struct {
	AdID     string
	Title    string
	Duration int32
	AssetURL string
	AssetMime string
	Width    int32
	Height   int32

	ImpTrackers     []string
	ClickID         string

	// Platform-specific (iQiyi VAST compliance)
	ClickType       string              // "0"/"4"/"11"/"14"/"15"/"67"
	ClickTrackers   []string            // click monitoring URLs (separate from landing URL)
	ClickThroughURL string              // landing page URL in ClickThrough CDATA
	TrackingEvents  map[string][]string // "firstQuartile" → urls, "complete" → urls, etc.
	DeeplinkApp     string              // APK package name (for type=14)
	IconURL         string              // DSP logo 25x25 PNG

	// Deprecated: kept for backward compatibility, use ClickThroughURL instead
	ClickTracker string
}

func BuildVast(params AdParams) (string, error) {
	duration := fmt.Sprintf("00:00:%02d", params.Duration)

	v := VAST{
		Version: "3.0",
		Ads: []Ad{{
			ID: params.AdID,
			InLine: InLine{
				AdSystem: "OpenDSP",
				AdTitle:  params.Title,
				Creatives: Creatives{
					Creative: []Creative{{
						Linear: Linear{
							Duration: duration,
							MediaFiles: MediaFiles{
								MediaFile: []MediaFile{{
									Delivery: "progressive",
									Type:     params.AssetMime,
									Width:    params.Width,
									Height:   params.Height,
									URL:      CDATA(params.AssetURL),
								}},
							},
						},
					}},
				},
			},
		}},
	}

	creative := &v.Ads[0].InLine.Creatives.Creative[0].Linear

	// Impression trackers (multiple nodes, parallel)
	for _, tracker := range params.ImpTrackers {
		if tracker != "" {
			url := tracker
			if params.ClickID != "" {
				url = appendQueryParam(url, "click_id", params.ClickID)
			}
			v.Ads[0].InLine.Impression = append(v.Ads[0].InLine.Impression, CDATA(url))
		}
	}

	// ClickThrough (landing URL + type attribute)
	ctURL := params.ClickThroughURL
	if ctURL == "" {
		ctURL = params.ClickTracker
	}
	if ctURL != "" {
		if params.ClickID != "" {
			ctURL = appendQueryParam(ctURL, "click_id", params.ClickID)
		}
		if creative.VideoClicks == nil {
			creative.VideoClicks = &VideoClicks{}
		}
		creative.VideoClicks.ClickThrough = &ClickThrough{
			Type:  params.ClickType,
			CDATA: CDATA(ctURL),
		}
	}

	// ClickTracking (separate monitoring URLs)
	for _, tracker := range params.ClickTrackers {
		if tracker != "" {
			url := tracker
			if params.ClickID != "" {
				url = appendQueryParam(url, "click_id", params.ClickID)
			}
			if creative.VideoClicks == nil {
				creative.VideoClicks = &VideoClicks{}
			}
			creative.VideoClicks.ClickTrackings = append(
				creative.VideoClicks.ClickTrackings,
				ClickTracking{CDATA: CDATA(url)})
		}
	}

	// TrackingEvents (firstQuartile, midpoint, thirdQuartile, complete, close, etc.)
	if len(params.TrackingEvents) > 0 {
		events := &TrackingEvents{}
		for event, urls := range params.TrackingEvents {
			for _, u := range urls {
				if u != "" {
					events.Trackings = append(events.Trackings, Tracking{
						Event: event,
						CDATA: CDATA(u),
					})
				}
			}
		}
		creative.TrackingEvents = events
	}

	// Icon (DSP Logo, 25x25 PNG)
	if params.IconURL != "" {
		v.Ads[0].InLine.Icon = &Icon{
			Program: "ADX",
			Width:   25,
			Height:  25,
			StaticResource: StaticResource{
				CreativeType: "image/png",
				CDATA:        CDATA(params.IconURL),
			},
		}
	}

	output, err := xml.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return xml.Header + string(output), nil
}

func appendQueryParam(url, key, value string) string {
	if strings.Contains(url, "?") {
		return url + "&" + key + "=" + value
	}
	return url + "?" + key + "=" + value
}
