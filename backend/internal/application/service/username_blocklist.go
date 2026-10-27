package service

import (
	"slices"
	"strings"
)

// Usernames are public (/public/{username}); offensive or glorifying names and ones that pretend to be
// the operator are rejected. The check is deliberately rough: it sees through digit substitutions (h1tl3r) and underscores.

// blockedSubstrings are terms that appear in no harmless name.
var blockedSubstrings = []string{
	// Nazism, racism, glorification of violence
	"hitler", "nazi", "goebbels", "himmler", "mengele", "eichmann", "heydrich",
	"siegheil", "heilhitler", "whitepower", "whitesupremac", "holocaust",
	"genocide", "kkk", "stalin", "polpot", "mussolini", "binladen", "alqaeda",
	"terrorist", "rapist", "paedo", "pedophile", "childmolest",
	// Insults
	"nigger", "nigga", "faggot", "tranny", "wetback", "retard",
	// Obscenities
	"fuck", "shit", "cunt", "bitch", "whore", "slut", "pussy", "asshole",
	"dildo", "porn", "blowjob", "cumshot",
}

// blockedWords occur inside harmless names (spice, raccoon, pakistan)
// and therefore only count as a whole name.
var blockedWords = []string{
	"spic", "kike", "gook", "chink", "coon", "paki", "fag", "rape", "cum", "isis",
}

// reservedNames could pretend to be the operator or services. They also count with digits,
// underscores and additions like "the_" or "_official".
var reservedNames = []string{
	"admin", "administrator", "sysadmin", "root", "system", "support", "staff",
	"moderator", "mod", "owner", "official", "help", "helpdesk", "security",
	"postmaster", "webmaster", "hostmaster", "noreply", "api", "www", "null",
	"undefined", "anonymous", "everyone", "travelblog",
}

var (
	reservedPrefixes = []string{"the", "real", "official", "mr", "ms", "im", "iam"}
	reservedSuffixes = []string{"official", "team", "hq", "real", "bot", "dev"}
)

var leetReplacer = strings.NewReplacer(
	"0", "o", "1", "i", "3", "e", "4", "a", "5", "s", "7", "t", "8", "b", "_", "",
)

// isReserved checks the name without digits and underscores, also in digit
// spelling (4dmin), after removing additions.
func isReserved(name string) bool {
	plain := strings.Map(func(r rune) rune {
		if r == '_' || (r >= '0' && r <= '9') {
			return -1
		}
		return r
	}, name)
	for _, form := range []string{plain, strings.ReplaceAll(leetReplacer.Replace(name), "_", "")} {
		for changed := true; changed; {
			if slices.Contains(reservedNames, form) {
				return true
			}
			changed = false
			for _, pre := range reservedPrefixes {
				if rest, ok := strings.CutPrefix(form, pre); ok && rest != "" {
					form, changed = rest, true
					break
				}
			}
			for _, suf := range reservedSuffixes {
				if rest, ok := strings.CutSuffix(form, suf); ok && rest != "" {
					form, changed = rest, true
					break
				}
			}
		}
	}
	return false
}

// checkUsernameAllowed rejects names with offensive terms; the name
// must be normalized already.
func checkUsernameAllowed(name string) error {
	folded := leetReplacer.Replace(name)
	rejected := newError(ErrInvalid, "username_not_allowed", "this username is not allowed")
	// Numeric codes of the neo-Nazi scene
	if strings.Contains(strings.ReplaceAll(name, "_", ""), "1488") || isReserved(name) {
		return rejected
	}
	for _, w := range blockedSubstrings {
		if strings.Contains(folded, w) {
			return rejected
		}
	}
	for _, w := range blockedWords {
		if folded == w {
			return rejected
		}
	}
	return nil
}
