# TODOs

---

## what i feel like doing

- just got my salary, feels like a great time to see what i need minimally, prioritize how i plan my month

- im leaning to transactions. 
- at least, inputting income to know inferred expenses.
- then expand the dashboards and reports from there.
- and then parent child expenses next.
- or budgets.... cuz expenses are not directly linked to budgets...??

---

## design TODOs

- rename to steph
- design the help popup UI

---

## feature TODOs

---

## enhancement TODOs

- reports should add month start/end, so can calculate the cashflow for that month
- reports should include these trends but for longer duration too, for a more holistic view, probably in lists form

---

## non-critical bug fix TODOs

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
    - tried automating inputting bank statements, eh, PDFs are complex, and still cant answer my questions

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
    - with accounts as your anchor, tbh you already have a pretty good idea of where u are now, how u came to be, and generally what you want to do next
    - then you decide if you want to dig more into the past => transactions
        - inputting your income => get your inferred expenses
        - inputting parent expenses => explains inferred expenses
        - inputting child expenses => even more granular explanation for parent expenses
    - or decide you want to be more future facing and thinking what to do tmr => budgets
        - so you know, your total balance, is not the whole story, there is some money you need to set aside
        - best if, "every dollar has a job", every dollar set aside in an "envelope" for a reason
        - if you did transactions first, you may have a better idea of a "starting point" for your budgets
        - just do keep in mind about reconciling the budgets later if just doing account snapshots
            - basically... u are allowed to "guess" or "deallocate" money from envelopes, without explicitly "linking" them to actual concrete transactions
            - so u can still answer the question "how much can i spend at the supermarket" even if you dont know how much you actually spent at the supermarket this past week!
        - advanced: goals and default allocations
    - further: money owed
        - like budgets, this also affects the money you actually have for spending

- usecase:
    - reuse existing aggregations (bank/octopus spending summaries and categories)
        - instead of inputting individual transaction, can simply keep track of one parent and edit date to today and new amount

- should put explanations in the help for each screen, explaining rationale behind different choices

- apparently this is a thing: Forward-Filling (Step Interpolation), that i sort of re-invented

- an interesting analogy of only having bank account snapshots is that we're treating bank account like stock price

- start/end and high/low is apparently re-inventing OHLC (open high low close) candle charts in investing

- tags like github
    - https://docs.github.com/en/enterprise-cloud@latest/issues/tracking-your-work-with-issues/using-issues/filtering-and-searching-issues-and-pull-requests#about-search-terms

