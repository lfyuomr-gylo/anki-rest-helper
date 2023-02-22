#!/usr/bin/env zsh

set -e

WORD=$1

# sd-conjugate -- see https://github.com/librehat/sdapi
# jq -- see https://stedolan.github.io/jq/
sd-conjugate $WORD | jq '[
    {set_field_if_not_empty: {IndicativePresentYo:        .[] | select(.pronoun == "yo") | select(.paradigm == "presentIndicative").word}},
    {set_field_if_not_empty: {IndicativePresentTu:        .[] | select(.pronoun == "tú") | select(.paradigm == "presentIndicative").word}},
    {set_field_if_not_empty: {IndicativePresentEl:        .[] | select(.pronoun == "él/ella/Ud.") | select(.paradigm == "presentIndicative").word}},
    {set_field_if_not_empty: {IndicativePresentNosotros:  .[] | select(.pronoun == "nosotros") | select(.paradigm == "presentIndicative").word}},
    {set_field_if_not_empty: {IndicativePresentVosotros:  .[] | select(.pronoun == "vosotros") | select(.paradigm == "presentIndicative").word}},
    {set_field_if_not_empty: {IndicativePresentEllos:     .[] | select(.pronoun == "ellos/ellas/Uds.") | select(.paradigm == "presentIndicative").word}},

    {set_field_if_not_empty: {ImperativeAffirmativeTu:    .[] | select(.pronoun == "tú")  | select(.paradigm == "imperative") | select(.tense == "affirmative").word}},
    {set_field_if_not_empty: {ImperativeAffirmativeUsted: .[] | select(.pronoun == "Ud.") | select(.paradigm == "imperative") | select(.tense == "affirmative").word}},

    {set_field_if_not_empty: {PreteriteYo:        .[] | select(.pronoun == "yo") | select(.paradigm == "preteritIndicative").word}},
    {set_field_if_not_empty: {PreteriteTu:        .[] | select(.pronoun == "tú") | select(.paradigm == "preteritIndicative").word}},
    {set_field_if_not_empty: {PreteriteEl:        .[] | select(.pronoun == "él/ella/Ud.") | select(.paradigm == "preteritIndicative").word}},
    {set_field_if_not_empty: {PreteriteNosotros:  .[] | select(.pronoun == "nosotros") | select(.paradigm == "preteritIndicative").word}},
    {set_field_if_not_empty: {PreteriteVosotros:  .[] | select(.pronoun == "vosotros") | select(.paradigm == "preteritIndicative").word}},
    {set_field_if_not_empty: {PreteriteEllos:     .[] | select(.pronoun == "ellos/ellas/Uds.") | select(.paradigm == "preteritIndicative").word}}
]'
