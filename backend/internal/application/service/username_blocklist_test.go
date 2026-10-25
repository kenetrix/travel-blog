package service

import (
	"errors"
	"testing"
)

func TestCheckUsernameAllowed(t *testing.T) {
	for _, name := range []string{
		"hitler", "adolf_hitler", "h1tl3r", "xx_n4zi_xx", "heil_hitl3r", "user1488",
		"fuck_you", "sh1t", "niggerx", "spic", "kkk_fan", "s_c_u_n_t", "pedophile99",
		"admin", "admin123", "4dmin", "the_admin", "admin_official", "real_support",
		"root", "moderator_1", "travel_blog", "travelblog_team", "official",
	} {
		if err := checkUsernameAllowed(name); !errors.Is(err, ErrInvalid) {
			t.Errorf("%q allowed: %v", name, err)
		}
	}
	for _, name := range []string{
		"alice", "bob_travels", "nigeria_fan", "pakistan", "spicy_food", "raccoon",
		"torpedo", "grape", "essex", "peacock", "user2024", "ana_88", "badminton",
		"stafford", "rooted", "mother", "administrate", "supporter", "systemd_fan",
	} {
		if err := checkUsernameAllowed(name); err != nil {
			t.Errorf("%q rejected: %v", name, err)
		}
	}
}
