package export

import (
	"math/rand"
)

var adjectives = []string{
	"able", "aged", "airy", "apt", "avid",
	"bare", "bold", "born", "busy", "calm",
	"cool", "cozy", "dark", "dear", "deep",
	"deft", "dim", "dry", "dull", "each",
	"easy", "even", "fair", "far", "fast",
	"fine", "firm", "flat", "fond", "free",
	"full", "glad", "gold", "good", "gray",
	"grim", "half", "hard", "high", "holy",
	"huge", "idle", "keen", "kind", "last",
	"late", "lazy", "lean", "left", "live",
	"long", "lost", "loud", "lush", "main",
	"mere", "mild", "mint", "mute", "neon",
	"next", "nice", "numb", "odd", "opal",
	"open", "oval", "pale", "past", "pink",
	"plan", "plum", "pure", "quad", "rare",
	"raw", "real", "red", "rich", "ripe",
	"rose", "ruby", "safe", "sage", "shy",
	"slim", "slow", "snug", "soft", "some",
	"sour", "sure", "tame", "tidy", "tiny",
	"top", "trim", "true", "twin", "used",
	"vast", "warm", "wavy", "wee", "wide",
	"wild", "wise", "wry", "zany", "zero",
}

var nouns = []string{
	"ant", "ape", "bat", "bear", "bee",
	"bird", "boar", "bull", "calf", "cat",
	"clam", "cod", "colt", "crab", "crow",
	"deer", "dog", "dove", "duck", "eel",
	"elk", "emu", "ewe", "fawn", "fish",
	"fly", "fox", "frog", "gnat", "goat",
	"gull", "hare", "hawk", "hen", "hog",
	"ibis", "jay", "kit", "kiwi", "lark",
	"lion", "lynx", "mare", "mink", "mole",
	"moth", "mule", "newt", "okra", "oryx",
	"otter", "owl", "pard", "pony", "puma",
	"quail", "ram", "rat", "rook", "seal",
	"slug", "swan", "toad", "trout", "tuna",
	"vole", "wasp", "worm", "wren", "yak",
}

// RandomName returns a human-readable random identifier like "bold-keen-fox".
func RandomName() string {
	a1 := adjectives[rand.Intn(len(adjectives))]
	a2 := adjectives[rand.Intn(len(adjectives))]
	n := nouns[rand.Intn(len(nouns))]
	return a1 + "-" + a2 + "-" + n
}
