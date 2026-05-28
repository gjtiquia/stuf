# TODOs

---

## design TODOs

- rename to steph
- design the help popup UI
- tags should allow "/" in it, for some sort of hierarchy / category of tags, for future query

---

## enhancement TODOs

- reports should add month start/end, so can calculate the cashflow for that month
- reports should include these trends but for longer duration too, for a more holistic view, probably in lists form

- dashboard

```
# stuf

as-of       : 2026-05-28 [!]

total       : HKD  20,288.28
budgeted    : HKD       0.00

on-budget net changes
2026-05     : HKD (43,835.96)
2026-04     : HKD (82,334.60)
2026-03     : HKD (64,843.42)

on-budget high to lows
2026-05     : HKD (46,559.95)
2026-04     : HKD (82,334.60)
2026-03     : HKD (64,843.42)

on-budget lows
2026-05     : HKD   4,378.55
2026-04     : HKD   4,378.55
2026-03     : HKD   4,378.55

you owe ppl : HKD       0.00
ppl owe you : HKD       0.00
```

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
    - first input first 3 months (or months from beginning of the year)
        - start and end
            - at the very least can know the cashflow status, monthly net growth/loss
        - highs and lows
            - from here, can start to know if there's a growing trend, especially the low to low trend, the safety net
        - notable changes
            - big drops from accounts, big gains, big transfers
    - can backfill later anytime, can be as simple as beginning of each month
    - do prioritize having the parent account finish before the child
        - as the parent is the anchor, tho can still be inferrred from child
        - however, it is probably more tedious to do a pass over all children, than a pass over the parent (if available) which includes all differences from child
    - then check out dashboard, reports from just the accounts
    - then input income via transactions

- usecase:
    - reuse existing aggregations (bank/octopus spending summaries and categories)
        - instead of inputting individual transaction, can simply keep track of one parent and edit date to today and new amount

- should put explanations in the help for each screen, explaining rationale behind different choices


