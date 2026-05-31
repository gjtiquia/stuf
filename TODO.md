# TODOs

---

## what i feel like doing

- just got my salary, feels like a great time to see what i need minimally, prioritize how i plan my month
- did budgets, experience was well (see below)! forward facing, instead of transactions which analyzes the past
- but came across the oweing money part and its conflicts with budgeting
- comes in two flavors
    - you owe ppl
        - should deduct available money for budgeting
        - but at the same time... especially for recurring expenses, should be part of the budgets
        - stuff that i pay first fully and is paid back later, i should budget the full amount
        - stuff that others pay fully and i pay back later, i should budget my share to pay back
        - hence two questions
            - how to track when ppl pay me back?
                - see "ppl owe you" overhaul below
            - how to track when i paid others?
                - reflected in account snapshots, but i cant see the actual "row", hard to check if the account snapshots actually includes my payment
                - see transactions below
    - ppl owe you
        - overhauled the original model, and used "ledgers" and transactions to represent ppl oweing and ppl returning money
        - decoupled from everything else, really serves as just a "reminder" for keeping track of where money is.
        - not a priority now because, the item i pay first is budgeted anyways, so its documented, i can always add this tracking information back later on
- transactions
    - transactions explains account balances
    - hence, transactions are a good place to document when and how did i pay back others
    - especially with tags and queries and such
    - would be great to start dealing with transactions!
    - then can try doing the inferred expenses, which makes reports much more interesting

- an edge case i eventually need to handle is... when im with my wife, i should be doing our finances tgt. 
- when back to long D, i need to be able to manage my own (while somewhat keeping track of hers)
- likely solved with some queries and tags and filtering...

---

## priotity

### budgets
- no need to show table header on each category, becomes repetitive under the limited screen space
- "transfer to" support during allocation
- goals (amount, date) and default monthly allocations

### formulas
- support app-wide formula support for all amounts

---

## design TODOs

- rename to steph
- design the help popup UI

---

## critical bug fix TODOs


### budgets
- budgets currency should only be relevant for the /budgets/list and the parenthesis, but allocations should allow multi currency!

---

## feature TODOs

### dashboard
- definitely useful to see the available money in the dashboard, feeling like doing mental math and, seeing "total" on the very top is kind of a lie
- would be useful to have some sort of high level multi-line notes, anchoring on like, the current plan for the month, or the current situation, and seeing the history of these notes would be valuable as well

### owe
- should implement soon, such that the available balance is accurate
    - tho its mainly the money i owe others that should deduct from available (kinda like a credit card)
    - for those that owe me, doesnt contribute really, i should be budgeting the full amount im paying, and the amount they owe me does not add to available, its more of tracking what money that is not yet returned

### currencies
- the more i think about it... starting to make sense to have historical currencies... but mainly useful for past analysis only

---

## enhancement TODOs

### budgets
- entering on a budget should directly go to... allocation list..? tho accounts goes to account details first... hmmm...
- should show total, budgeted, and available when doing allocations
- other that "set total", "add", "remove", should also have
    - allocate all available
    - reconcile available (makes negative available zero)

### formulas
- once formulas are implemented in owed amounts, they should be... widely available wherever you can input amounts?

---

## non-critical bug fix TODOs

### accounts
- edit account name insertion typing, cursor keeps going to the end... super annoying in mass renaming
- edit balance amount cannot move cursor to insert
- edit negative balance amount, type "-" should not immediately add 0 at the end

### reports
- should not scroll left/right further than what snapshots are available

### budgets
- category input super buggy
- sorting should be from newest on top

---

## discarded TODOs

- clipping when text is above the terminal size!
    - discarded because the alternative is managing scolling within the TUI which would be complex
    - better deal with it via better pagination

---

## docs TODOs

- personal story:
    - stopped tracking for a few days
    - constantly playing catchup
    - every end of the month need to input EVERY SINGLE transaction
    - blows up in complexity when handling expenses for both wife and i, shared accounts, money owed
    - especially after wedding and trips, going back and forth, hectic lifestyle
    - tried automating inputting bank statements, eh, PDFs are complex, and still cant answer my questions
    - also... moving around back and forth, wedding, lots of sudden events, analyzing the past becomes somewhat meaningless as there's no reliable spending pattern
    - hence a lot of common budgeting tools sort of... break down, as the assumptions they had for building those kind of broke down as well
    - also "explaining" past expenses seems to "justify"... and that doesnt drive better actions. the "explanation" should serve to make better future decisions, not make myself feel better

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
    - do try to resist the urge to explain everything in the past
        - its only helpful if it helps with future actions
        - be ok with "explained enough", know "enough" to act and plan
        - the less u explain the past, the less ur future self will be pressured into analyzing EVERYTHING
        - reduce friction for managing ur expenses

- usecase:
    - reuse existing aggregations (bank/octopus spending summaries and categories)
        - instead of inputting individual transaction, can simply keep track of one parent and edit date to today and new amount

- should put explanations in the help for each screen, explaining rationale behind different choices

- apparently this is a thing: Forward-Filling (Step Interpolation), that i sort of re-invented

- an interesting analogy of only having bank account snapshots is that we're treating bank account like stock price

- start/end and high/low is apparently re-inventing OHLC (open high low close) candle charts in investing

- tags like github
    - https://docs.github.com/en/enterprise-cloud@latest/issues/tracking-your-work-with-issues/using-issues/filtering-and-searching-issues-and-pull-requests#about-search-terms

---

## research TODOs

"Frictionless Tracking" / Flow-Based Budgeting:
Look up blogs on "anti-budgets" or "flow-based budgeting."
They argue that tracking net-worth snapshots (like your balance anchors) is the only reliable way to measure financial health,
while transaction tracking is just behavioral therapy.

Read up on Paula Pant (Afford Anything). 
She heavily promotes the "Anti-Budget." 
The core thesis is that willpower is finite. 
If you hit your 20% savings goal on payday, 
it mathematically does not matter if you spend the remaining 80% on groceries, home improvement tools, or a spontaneous trip to Tokyo. 
You just look at your checking account balance, and if it's above zero, you're good.
The core principle is "Save first, spend the rest." 
This is a mathematical, not behavioral, approach. 
It acknowledges that categorizing coffee vs. groceries is essentially "ledger-making" for the sake of anxiety, not for the sake of wealth building.

Look up old UI/UX teardowns of an app called Level Money (circa 2015). 
It did exactly this: it calculated your income, subtracted fixed expenses, 
and gave you a single "Spendable" number for the day, week, and month. 
It was brilliant, widely loved, bought by Capital One, and then killed. 
Many designers still cite it as the peak of budgeting UX because it entirely eliminated cognitive load.

The "Guilt-Free" Movement: 
Ramit Sethi's I Will Teach You To Be Rich advocates for automating the top-down flow so you can spend guilt-free. 
stuf is essentially the software manifestation of this book's philosophy.

Thaler — Mental Accounting
Thaler defines mental accounting as how individuals and households organize, evaluate, and keep track of financial activities. 
This is directly relevant to your budgets-as-proxy-accounts idea: users naturally create mental buckets anyway.

Galperti — A Theory of Personal Budgeting
Galperti analyzes budgeting as a tool for self-control, balancing commitment with flexibility under uncertainty. 
This is very relevant because your product is almost a “flexible commitment” system: enough structure to guide action, not so much structure that life breaks it.

Antonides et al. — Mental Budgeting and Household Finance
This research found mental budgeting is common and associated with having an overview of expenses/current accounts and household financial management. 
This supports the idea that budgeting is not just math; it is a cognitive organization tool.


---

## my personal thoughts

- after entering accounts balances, then the OHLC from reports, it gave a very "good enough" view of the current state of my finances
- after entering budgets, i have a pretty good idea of how much i should be able to "save" per month, how to lift up the monthly "low"
- and this is all before entering a single transaction!
- will test run for a month and see how it goes!

- started doing budgets
- its pretty nice cuz i can deduct an estimate from budgets as its not linked to transactions
- and if i forget to deduct, i can always freely "start over" and give myself a budget until the end of the month (or next income)
- and since its not "synced" to account balances, i added a "buffer" to transfer into, its like "used money", money i expect to zero out after i sync my account balance (which is expected to drop)
- this way i dun artificially inflate the available money, while still retaining the purpose of budget envelopes
- also added budget for "lowest-low" to never spend to raise the previous month's low, and a "leftover" for unexpected expenses (social occasions and whatnot) without being too granular with the budgets

- but also started doing transactions
- start to feel a bit of duplicate work...
    - perhaps... add two fields
    - deduct from budget
    - add to budget
    - both optional, so it automates "transferring from xxx budget to buffer budget"
- mostly using it to track money "i owe ppl", since we decided that the "ppl owe you" ledger should be only for money ppl owe you
    - track that i actually returned the money



