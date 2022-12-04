#!/usr/bin/env zsh

set -e

WORD=$1

# sd-conjugate -- see https://github.com/librehat/sdapi
# jq -- see https://stedolan.github.io/jq/
sd-conjugate $WORD | jq '{
    IndicativePresentYo:        .[] | select(.pronoun == "yo") | select(.paradigm == "presentIndicative").word,
    IndicativePresentTu:        .[] | select(.pronoun == "tú") | select(.paradigm == "presentIndicative").word,
    IndicativePresentEl:        .[] | select(.pronoun == "él/ella/Ud.") | select(.paradigm == "presentIndicative").word,
    IndicativePresentNosotros:  .[] | select(.pronoun == "nosotros") | select(.paradigm == "presentIndicative").word,
    IndicativePresentVosotros:  .[] | select(.pronoun == "vosotros") | select(.paradigm == "presentIndicative").word,
    IndicativePresentEllos:     .[] | select(.pronoun == "ellos/ellas/Uds.") | select(.paradigm == "presentIndicative").word,

    ImperativeAffirmativeTu:    .[] | select(.pronoun == "tú")  | select(.paradigm == "imperative") | select(.tense == "affirmative").word,
    ImperativeAffirmativeUsted: .[] | select(.pronoun == "Ud.") | select(.paradigm == "imperative") | select(.tense == "affirmative").word
}'
