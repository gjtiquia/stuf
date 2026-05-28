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

---

## bug fix TODOs

- negative amount parenthesis should clamp over the numbers
- edit account name insertion typing, cursor keeps going to the end... super annoying in mass renaming
- edit balance amount cannot move cursor to insert
- edit negative balance amount, type "-" should not immediately add 0 at the end

---

## docs TODOs

- personal story:
    - stopped tracking for a few days
    - constantly playing catchup
    - every end of the month need to input EVERY SINGLE transaction
    - blows up in complexity when handling expenses for both wife and i, shared accounts, money owed
    - especially after wedding and trips, going back and forth, hectic lifestyle

- onboarding
    - start with accounts
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
    - credit cards are accounts too
        - typically enter balance from each month statement as a negative balance
        - main purpose is more to make sure you know that you have money oweing, deducts from the total and available
        - trends may seem weird at first, but it makes sense
            - typically if entering balance, its like, constantly negative
            - if net change is positive, its good because it means the money you are borrowing is decreasing each month
            - it its still a transfer between accounts, from debit to credit, so it tells the whole story, the "hidden" trends
            - it could show something thats worse than it seems, or could show that its better than it seems
            - eg. it may look like you have a lot available, or the monthly lows are ok, but after adding credit card you may realize you are in debt for months
            - eg. it may look like a HUGE net loss, cuz paying credit card, but inputting credit card balance means its simply a large amount transfer and wont count to net change of all accounts
    - then check out dashboard, reports from just the accounts
    - then input income via transactions

- usecase:
    - reuse existing aggregations (bank/octopus spending summaries and categories)
        - instead of inputting individual transaction, can simply keep track of one parent and edit date to today and new amount

- should put explanations in the help for each screen, explaining rationale behind different choices


