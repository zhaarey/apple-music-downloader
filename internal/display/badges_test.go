package display

import "testing"

func TestBadges(t *testing.T) {
	tests := []struct {
		name                 string
		contentRating        string
		isAppleDigitalMaster bool
		isMasteredForItunes  bool
		want                 string
	}{
		{name: "explicit", contentRating: "explicit", want: "[E]"},
		{name: "clean", contentRating: "clean", want: "[C]"},
		{name: "unrated", contentRating: "", want: ""},
		{name: "apple digital master", isAppleDigitalMaster: true, want: "[M]"},
		{name: "mastered for itunes", isMasteredForItunes: true, want: "[M]"},
		{name: "explicit and master", contentRating: "explicit", isAppleDigitalMaster: true, want: "[E] [M]"},
		{name: "clean and master", contentRating: "clean", isMasteredForItunes: true, want: "[C] [M]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Badges(tt.contentRating, tt.isAppleDigitalMaster, tt.isMasteredForItunes)
			if got != tt.want {
				t.Errorf("Badges(%q, %t, %t) = %q, want %q",
					tt.contentRating, tt.isAppleDigitalMaster, tt.isMasteredForItunes, got, tt.want)
			}
		})
	}
}

func TestLabel(t *testing.T) {
	tests := []struct {
		name                 string
		nameText             string
		contentRating        string
		isAppleDigitalMaster bool
		isMasteredForItunes  bool
		want                 string
	}{
		{name: "no badges", nameText: "Song", want: "Song"},
		{name: "explicit", nameText: "Song", contentRating: "explicit", want: "[E] Song"},
		{name: "clean and master", nameText: "Song", contentRating: "clean", isAppleDigitalMaster: true, want: "[C] [M] Song"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Label(tt.nameText, tt.contentRating, tt.isAppleDigitalMaster, tt.isMasteredForItunes)
			if got != tt.want {
				t.Errorf("Label(%q, %q, %t, %t) = %q, want %q",
					tt.nameText, tt.contentRating, tt.isAppleDigitalMaster, tt.isMasteredForItunes, got, tt.want)
			}
		})
	}
}
