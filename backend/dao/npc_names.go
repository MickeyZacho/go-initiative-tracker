package dao

import (
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"
)

// NicknameSeparator sits between a copy's numbered name and its generated
// nickname: "Goblin 3 · Snagtooth".
const NicknameSeparator = " · "

// pickIndex chooses a random index below n. A package variable so tests can
// make nickname picks deterministic.
var pickIndex = func(n int) int { return rand.IntN(n) }

var goblinoidNames = []string{
	"Snagtooth", "Grimjaw", "Muckfist", "Skrag", "Ratgut", "Bonegnaw", "Snikkit",
	"Gutrot", "Lugbash", "Nagrub", "Krimp", "Yagnak", "Ugrok", "Grubnik", "Skabbit",
	"Mogrot", "Zugga", "Fleshrot", "Gitspit", "Durbag", "Kraknak", "Splug", "Boglug",
	"Nibbik", "Scabnose", "Murgol", "Rotfang", "Grizzik", "Ulgrom", "Skiv", "Blight",
	"Gazbat", "Throk", "Wretch", "Snarl", "Gulpa", "Krug", "Narzug", "Pikk", "Vrask",
}

var giantNames = []string{
	"Brumhild", "Gorm", "Thunk", "Hrolfgar", "Ymra", "Bolverk", "Grunda", "Skarn",
	"Thrud", "Ogmund", "Hulda", "Krag", "Bergelmir", "Torva", "Ragnhild", "Stomp",
	"Grimbold", "Hrung", "Ulfa", "Dunmar", "Gudrun", "Rokk", "Sigvor", "Bramm",
	"Ingrid", "Halvard", "Mogga", "Tusk", "Vagn", "Ottar",
}

var dragonNames = []string{
	"Vyraxis", "Kalthrax", "Ysvirith", "Morvathar", "Sesskarra", "Thalzurion",
	"Ixalith", "Drovanyx", "Velkathra", "Zyrrhax", "Auremyth", "Skarthion",
	"Nyxalara", "Ombrevex", "Cirraxus", "Pyrestra", "Valtharyn", "Qorvexis",
	"Halzidrae", "Xenthara",
}

var commonNames = []string{
	"Aldric", "Brenna", "Cedric", "Dara", "Edwin", "Fenna", "Garrick", "Helga",
	"Ivo", "Jessa", "Kael", "Lysa", "Merrick", "Nessa", "Osric", "Petra", "Quill",
	"Rowan", "Sabine", "Tobin", "Ulric", "Vera", "Wendel", "Yara", "Bram", "Corin",
	"Elsbeth", "Finn", "Greta", "Hollis", "Isolde", "Jory", "Kestrel", "Linnea",
	"Magnus", "Nadia", "Orla", "Perrin", "Renna", "Silas", "Thea", "Varis", "Wren",
	"Alaric", "Bryn", "Dorian", "Eira", "Gideon", "Maren", "Tamsin",
}

// descriptors stand in for names on creatures that wouldn't have one (beasts,
// undead, oozes, ...). They're picked to fit almost anything.
var descriptors = []string{
	"Scarred", "One-eyed", "Limping", "Hulking", "Scrawny", "Grizzled", "Pale",
	"Dark", "Ancient", "Snarling", "Notched", "Twitchy", "Lumbering", "Blackened",
	"Mottled", "Gaunt", "Crooked", "Bristling", "Silent", "Ragged", "Spiny",
	"Swollen", "Hunched", "Restless", "Patchy", "Broken-horned", "Stubby",
	"Glassy-eyed", "Jagged", "Tattered", "Bloated", "Wiry", "Burly", "Sly",
	"Grey", "Striped", "Blind", "Lame", "Shaggy", "Hissing",
}

// namedTypes are creature types (lower-case) that get personal names; every
// other known type gets a descriptor. "" is a template made in the app, which
// has no type, so it is treated as a person.
var namedTypes = map[string]bool{
	"": true, "humanoid": true, "giant": true, "dragon": true,
	"fey": true, "fiend": true, "celestial": true,
}

var goblinoidWords = []string{"goblin", "bugbear", "orc", "kobold", "gnoll"}

// nicknamePool picks the list a template's copies draw nicknames from, by
// template name first (so an app-made "Goblin" still gets goblin names), then
// by creature type.
func nicknamePool(templateName, creatureType string) []string {
	creatureType = strings.ToLower(strings.TrimSpace(creatureType))
	if !namedTypes[creatureType] {
		return descriptors
	}
	words := strings.Fields(strings.ToLower(templateName))
	for _, w := range words {
		for _, g := range goblinoidWords {
			if strings.HasPrefix(w, g) || strings.HasSuffix(w, g) {
				return goblinoidNames
			}
		}
	}
	// "Hill Giant" but not "Giant Rat": giant has to be the last word.
	last := ""
	if len(words) > 0 {
		last = words[len(words)-1]
	}
	switch {
	case creatureType == "giant" || last == "giant" || last == "ogre" || last == "troll" || last == "ettin":
		return giantNames
	case creatureType == "dragon" || strings.Contains(strings.ToLower(templateName), "dragon"):
		return dragonNames
	}
	return commonNames
}

// highestCopyNumber returns the largest copy number among names of earlier
// copies of templateName ("Goblin 4", "Goblin 4 · Snagtooth"). A plain
// "Goblin" (a lone first copy) counts as 1. Copies renamed to something else
// are ignored. Using the highest number rather than a count means removing a
// copy never leads to two "Goblin 3"s.
func highestCopyNumber(templateName string, names []string) int {
	re := regexp.MustCompile(`^` + regexp.QuoteMeta(templateName) + `(?: (\d+))?(?:` + regexp.QuoteMeta(NicknameSeparator) + `.*)?$`)
	highest := 0
	for _, n := range names {
		m := re.FindStringSubmatch(n)
		if m == nil {
			continue
		}
		num := 1
		if m[1] != "" {
			num, _ = strconv.Atoi(m[1])
		}
		highest = max(highest, num)
	}
	return highest
}

// usedNicknames collects the nicknames already taken in an encounter.
func usedNicknames(names []string) map[string]bool {
	used := map[string]bool{}
	for _, n := range names {
		if _, nick, ok := strings.Cut(n, NicknameSeparator); ok {
			used[nick] = true
		}
	}
	return used
}

// pickNickname draws an unused nickname from pool and marks it used. When the
// pool is exhausted it repeats one rather than failing.
func pickNickname(pool []string, used map[string]bool) string {
	free := make([]string, 0, len(pool))
	for _, n := range pool {
		if !used[n] {
			free = append(free, n)
		}
	}
	if len(free) == 0 {
		free = pool
	}
	nick := free[pickIndex(len(free))]
	used[nick] = true
	return nick
}
