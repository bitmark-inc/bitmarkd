package account_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/bitmark-inc/bitmarkd/account"
)

type item struct {
	base58Seed string
	phrase     string
}

var validItems = []item{
	{
		base58Seed: "9J879ykQwWijwsrQbGop819AiLqk1Jf1Z",
		phrase:     "hundred diary business foot issue forward penalty broccoli clerk category ship help",
	},
	{
		base58Seed: "9J878SbnM2GFqAELkkiZbqHJDkAj57fYK",
		phrase:     "file earn crack fever crack differ wreck crazy salon imitate swamp sample",
	},
	{
		base58Seed: "5XEECt18HGBGNET1PpxLhy5CsCLG9jnmM6Q8QGF4U2yGb1DABXZsVeD",
		phrase:     "accident syrup inquiry you clutch liquid fame upset joke glow best school repeat birth library combine access camera organ trial crazy jeans lizard science",
	},
	{
		base58Seed: "5XEECqxPZwQCBMACLXjT2ZLSkhrFqibTZSb1p1PAwSgqmEwaw46iRpt",
		phrase:     "about hurt rebel loan pattern water nose affair outside blouse color discover obey jealous portion penalty embrace fog move tool betray weird brother vanish",
	},
}

func TestValidBase58EncodedSeedToPhrase(t *testing.T) {
	for _, item := range validItems {
		phrase, err := account.Base58EncodedSeedToPhrase(item.base58Seed)
		if nil != err {
			t.Errorf("actual error: %s, expected no error", err)
		}

		actualPhrase := strings.Join(phrase, " ")
		if !reflect.DeepEqual(item.phrase, actualPhrase) {
			t.Errorf("actual phrase: %v, expected: %v", actualPhrase, item.phrase)
		}
	}
}
