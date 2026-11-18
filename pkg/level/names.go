package level

// English names for the built-in palettes, for the editor's English interface.
// Keyed by list and id; a test checks every palette entry has one.
var englishNames = map[string]map[string]string{
	"sky": {
		"day": "Day blue", "dusk": "Dusk orange", "night": "Night", "dawn": "Dawn pink",
		"overcast": "Overcast", "starry": "Starry",
	},
	"ground": {
		"brown": "Brown earth", "red": "Red earth", "gray": "Grey rock", "sand": "Sand",
		"frozen": "Frozen earth", "ash": "Volcanic ash",
	},
	"turf": {
		"green": "Meadow green", "autumn": "Autumn orange", "snow": "Snow white",
		"sand": "Desert yellow", "amethyst": "Amethyst", "ember": "Ember red",
	},
	"brick": {"red": "Red brick", "blue": "Blue brick", "stone": "Stone brick", "gold": "Gold brick"},
	"stair": {
		"gray": "Grey stone", "sand": "Sandstone", "slate": "Slate", "ice": "Ice",
		"obsidian": "Obsidian",
	},
	"pipe":  {"green": "Green", "blue": "Blue", "orange": "Orange", "gray": "Grey"},
	"theme": {
		"grassland": "Grassland", "dusk": "Dusk", "night": "Night", "dawn": "Dawn",
		"snow": "Snowfield", "desert": "Desert", "underground": "Underground", "lava": "Lava",
	},
}

// EnglishName is a palette entry's English name, or its id if it has none.
// list is one of sky, ground, turf, brick, stair, pipe, theme.
func EnglishName(list, id string) string {
	if n, ok := englishNames[list][id]; ok {
		return n
	}
	return id
}
