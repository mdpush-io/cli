package crypto

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

// Word list for passphrase generation — common, memorable, easy to type.
var passphraseWords = []string{
	"amber", "anchor", "arctic", "autumn", "azure",
	"beacon", "bloom", "brave", "breeze", "bright",
	"canyon", "cedar", "chrome", "cipher", "cliff",
	"coral", "crane", "crest", "crystal", "cycle",
	"dagger", "dawn", "delta", "desert", "drift",
	"eagle", "echo", "ember", "epoch", "fable",
	"falcon", "flame", "flint", "forest", "frost",
	"galaxy", "ghost", "glacier", "granite", "grove",
	"harbor", "haven", "hollow", "horizon", "hunter",
	"indigo", "iron", "ivory", "jade", "jasper",
	"kindle", "lance", "lantern", "lava", "lunar",
	"maple", "marble", "marsh", "meadow", "mesa",
	"mirror", "mist", "moss", "mountain", "nebula",
	"nimbus", "noble", "north", "nova", "oak",
	"ocean", "onyx", "orbit", "osprey", "oxide",
	"pebble", "phoenix", "pine", "pixel", "prism",
	"quartz", "raven", "reef", "ridge", "river",
	"sage", "scarlet", "shadow", "silver", "slate",
	"solar", "spark", "spire", "storm", "summit",
	"thunder", "tiger", "timber", "torch", "tower",
	"twilight", "vapor", "venture", "violet", "vortex",
	"wander", "willow", "winter", "wolf", "zenith",
}

// GeneratePassphraseSuggestion generates a memorable passphrase suggestion
// in the format: word-word-word-number (e.g., "violet-mountain-echo-1987").
// Uses crypto/rand for all random selections.
func GeneratePassphraseSuggestion() (string, error) {
	parts := make([]string, 3)
	for i := range 3 {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(passphraseWords))))
		if err != nil {
			return "", fmt.Errorf("generating passphrase: %w", err)
		}
		parts[i] = passphraseWords[idx.Int64()]
	}

	// Generate a 4-digit number (1000-9999)
	num, err := rand.Int(rand.Reader, big.NewInt(9000))
	if err != nil {
		return "", fmt.Errorf("generating passphrase number: %w", err)
	}
	parts = append(parts, fmt.Sprintf("%d", num.Int64()+1000))

	return strings.Join(parts, "-"), nil
}
