# TODOs

---

## design TODOs

- rename to steph
- design the help popup UI

---

## enhancement TODOs

- dashboard

```
net change to today
from may start  : HKD 
from may high   : HKD 
from apr high   : HKD 

recent months
may high to low : HKD
apr high to low : HKD
mar high to low : HKD

high to high trends
apr to may      : HKD
mar to apr      : HKD
feb to mar      : HKD

low to low trends
apr to may      : HKD
mar to apr      : HKD
feb to mar      : HKD
```

- reports should include these trends but for longer duration too, for a more holistic view, probably in lists form

---

## bug fix TODOs

- negative amount parenthesis should clamp over the numbers
- edit account name insertion typing, cursor keeps going to the end... super annoying in mass renaming
- edit balance amount cannot move cursor to insert

---

## docs TODOs

- personal story:
    - stopped tracking for a few days
    - constantly playing catchup
    - every end of the month need to input EVERY SINGLE transaction
    - blows up in complexity when handling expenses for both wife and i, shared accounts, money owed
    - especially after wedding and trips, going back and forth, hectic lifestyle

- onboarding
    - first input first 3 months (or months from beginning of the year), start and end -> highs and lows -> notable changes
    - can backfill later anytime, can be as simple as beginning of each month
    - do prioritize having the parent account finish before the child
        - as the parent is the anchor, tho can still be inferrred from child
        - however, it is probably more tedious to do a pass over all children, than a pass over the parent (if available) which includes all differences from child
    - then check out dashboard, reports

- usecase:
    - reuse existing aggregations (bank/octopus spending summaries and categories)
        - instead of inputting individual transaction, can simply keep track of one parent and edit date to today and new amount

- should put explanations in the help for each screen, explaining rationale behind different choices


