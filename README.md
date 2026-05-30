# stuf

> ehh... apparently this name is taken... gotta think of a new project name...
> perhaps kuka, kaku, kunga, kwunka, ggaa, gungaa (管家)
> or steph!

```
- [stu]ward [f]inance
- [stuf]f
```

a finance tool

app promise
- balance snapshots anchor the truth
- detailed records can be incomplete
- users should be able to go fast without fear
- undo/history makes mutations safe
- fresh balances can re-anchor messy records
- backups and sqlite access prevent lock-in
- the app should feel guilt-free, not like bookkeeping homework

## the idea

- most finances are "bottom up"
- miss one transaction, u sort of f-ed up
- this is designed to be "top down"
- at a minimal level, even just entering ur monthly bank balance should give u some sort of analysis
- and then from the missing information, the app should implicitly guide u to answer more questions about ur own finances, hence u will only look for answers that matter
- the hope is that, even filling in maybe 40-70% of information, is enough to give u enough knowledge and control about ur finances, ur cash flow
- that also alleviates the pressure of keeping up-to-date and perfect records
- ideally this also incorporates great queries, note taking, and zero-based envelop budgeting
- ideally the envelops are not tied to "months", they can sort of carry over by default, kind of "global", like there only ever is one envelop
- essentially combining Google Sheets / Excel Spreadsheets together with Actual Budget / YNAB, with the power of SQL queries

answers that `stuf` should be able to answer:
- how many months until i can save up till x amount?
- how much can be saved per month?
- how much can i invest per month, while saving money, without going broke, while still be able to travel?
- how much money can i use now at the supermarkey?
- can i afford this thing?
- net growth or loss last month / last few months / last year? why?
- how much money do i need to allocate monthly for x? (yearly expense, saving goals, investments, emergency fund)
- what is my current strategy / action plan for my finances?
- how much money did i bleed for the past few months? can my income offset it? what about... future yearly / tax / big expenses into account?

questions behind the numbers

balance truth questions
- did i end the month with more or less money?
- how much did my on-budget money actually change?
- are my assets growing over time, even if this month felt expensive?
- is my day-to-day money stable, or slowly draining?
- is my current drop normal compared to previous months?

income / expense questions
- how much money came in this month?
- if my balances changed by x and income was y, how much must have gone out?
- how much did i really bleed this month?
- is my income enough to cover my actual burn?
- am i spending more than i earn, or just feeling broke because of timing?
- how much of the inferred expense have i explained with transactions?
- what expenses are still unexplained?

liquidity stress questions
- how far did my money fall from its monthly high?
- what was the worst balance dip i had to survive?
- did this month require more buffer than usual?
- was this month expensive, or did income just arrive at an awkward time?
- what was the bigger load: total expenses, or the worst intra-month drawdown?
- how much money did i need available to not feel squeezed?

trend questions
- are my monthly highs going up or down?
- is my peak capacity growing?
- are my monthly lows going up or down?
- is my safety floor improving?
- even if i spend a lot each month, am i recovering to higher highs?
- are my bad moments getting better or worse?
- is the normal low point of my finances becoming safer?

planning questions
- based on my recent burn, how much buffer do i need?
- based on my recent liquidity stress, how much should stay on-budget?
- how much can i move to investments without making day-to-day money unstable?
- how much can i save per month after accounting for real expenses?
- can i afford a yearly / tax / travel expense without breaking my safety floor?
- what monthly allocation would make future big expenses boring instead of scary?

metric meanings
- net growth / change = did i end with more or less money?
- income = how much came in?
- derived expenses = how much must have gone out?
- high-to-low = how far did balances fall inside the month?
- high-to-high = are my best moments getting better?
- low-to-low = are my worst moments getting safer?

things that `stuf` should be able to support
- accounts (on-budget and off-budget)
- multi-currency
- shared/local household setup
- zero-based / envelop budgeting
    - being one month ahead (or more)
- tagging
- queries
- aggregation (sum, count)
- notes
- reports (cash flow / category trend)
- exporting (prevent lock-in)
- shared/owed expense tracking (shared expense with room mate, helping friend/family pay and waiting for them to pay back)
- undo support
    - any mutations should be reversible or at least editable and not lossy, at least for that session
- backup support
    - easy to backup and restore
    - "git" / "version control" mindset

stretch goals of `stuf`
- track investments and their performance
- debts (student loans)

outcomes of using `stuf`
- can answer any of the questions above on a daily basis (almost immediate basis)

## the app

- will develop as a TUI first, for my personal use, and to quickly verify that the workflow works
- also as a TUI, forced to think of things in an efficient way, vim-style keyboard shortcuts
- open to making it a web app as well in the future

basic top down flow
- monthly bank statement balances -> net change / growth context
- monthly income -> net cash flow in/out
- lump sum (eg. credit card payment) -> cash flow out sources, percentage of expense, tagging
- transactions -> tagging and deeper analysis; parent-child transaction trees explain lump sums without double counting

dashboard net change
- use net change, not net growth, for the current month dashboard number
- growth has an expectation of growth baked into the word
- but month-to-date movement is often negative in a completely normal way
- income usually arrives first, then expenses come after, whether income lands at the beginning, middle, or end of the month
- so the dashboard should not make normal spending feel like failure
- it should answer: how did i do so far this month?
- the current month needs reference points so i can tell whether the expected drop is larger than expected, about the same, or better than before
- previous month drops give more context for what normal spending looks like
- longer month-to-month trends show whether assets are still growing despite the spending inside each month
- the dashboard net change should focus on on-budget accounts by default
- this makes it about on-budget balance rhythm, not the full asset picture
- the dashboard should be compact enough to answer quick glance questions:
  how fresh is my data? what is my current total? what changed recently? how bad were the drawdowns? where is my floor?
- dashboard `as-of` means the latest relevant balance snapshot date, not today's date
- show `[!]` beside dashboard `as-of` when that date is older than the app's current date
- this makes stale balance truth visible before the dashboard shows any totals
- healthy on-budget movement can mean growing or staying roughly flat
- flat can be good if surplus is intentionally moved to investments / off-budget accounts
- the dashboard mostly answers whether day-to-day money is stable, growing, or dropping within the usual range
- broader asset growth belongs in reports, where off-budget / investment accounts can add context without making the home screen noisy
- the most useful balance snapshots are probably beginning of month, end of month, highs, and lows
- then should add snapshots that are significant, like, balance on receive income, balance on paying rent (big expense), big transfers
- daily balance snapshots would be great too, cuz more info = more insight, but following the app philosophy, its optional
- but even just those key snapshots basically works, because the app is designed around useful anchors instead of perfect bookkeeping
- if an account has a first known balance, missing history before that should be treated as flat from that first known balance, not as zero
- same idea after the latest known balance: carry it forward until another snapshot says otherwise
- only accounts with no balance snapshots at all should behave like zero
- boundary values should be as-of values: use the latest snapshot on or before the boundary, unless there is no earlier snapshot
- month highs and lows should include the carried month-start value plus snapshots inside that month
- this keeps the dashboard aligned with the app promise: balance snapshots anchor the truth, imperfect details are okay, and the app stays guilt-free

lazy reconciliation
- balance snapshots anchor everything
- detailed records can be incomplete without ruining macro analysis
- transactions, budgets, owed ledgers, and owed transactions explain or plan around balances
- transactions do not update balances
- transactions do not update budgets
- owed transactions do not update balances
- budget allocations do not update balances
- if things get messy, enter fresh balances and continue
- the app should feel guilt-free, not like bookkeeping homework

account / transaction analogy
- accounts are balance anchors
- transactions explain account balance movement
- budgets behave like proxy accounts for on-budget money
- budget allocations explain budget movement
- budgets are intentionally decoupled from transactions
- owed ledgers behave like lightweight virtual accounts for money people owe you
- owed transactions explain owed ledger movement
- account currency is the balance anchor currency
- budget currency is the proxy account anchor currency
- owed ledger currency is the proxy account anchor currency
- transaction currency is the event currency
- owed transaction currency is the event currency
- child transactions and owed transactions convert into their parent/proxy anchor currency for remaining/balance math

## the implementation 

stack
- golang, bubbletea, sqlite, goose, sqlc

keyboard shortcuts
- separate actions and keys
- vertical navigation uses up/down, j/k, and tab/shift-tab depending on screen
- horizontal navigation is contextual:
    - menus: left/h behaves like esc/back; right/l behaves like enter/open (yazi-style)
    - paginated selects: left/right changes page; h/l stays available for filter typing
    - text fields and active filters: h/l types normally; left/right moves caret or page
    - list-backed detail screens: left/h and right/l move to the previous/next item in the source list order
    - only show horizontal shortcuts when the action is available (hide at boundaries)
- ctrl+n is the canonical shortcut for creating/appending a new element from a list screen when that list has a matching create/add flow
- ctrl+e and ctrl+d are canonical shortcuts for editing/deleting the selected list element when supported
- forms launched from list shortcuts return to that list after successful submit
- plain letter keys stay available for list filters
- ctrl+s is the canonical shortcut for submitting a form immediately with currently committed form values

components
- custom components, dont fight with the defaults
- see if can write it kinda like react components
    - h1, newline, tables, text and formatted text (date/money)
    - styling = each item adjusts global width 
- the "hope" is that... when we make a web app these can translate better to semantic HTML

"url"
- show this to users to know how they got there
- also predictable esc (back) language
- logic also becomes easy to flow
    - dashboard can show depending on url
    - keyboard shortcuts can change depending on url
    - components can show depending on url

screen layout
- the URL is the boundary between context and page content
- context summaries render above the URL
- lists, forms, and actions render below the URL
- the renderer should enforce this order by default so new screens do not manually place URLs
- typical shape:
    - title
    - context summary, optional
    - `/path/`
    - main content, optional
    - options/actions, optional

mockup styling
- TUI mockups should read like fixed-width interfaces, not prose
- context and dashboard blocks should use visual tab stops
- align `:` across the whole visible context/content block when the rows belong to the same screen area
- align money on decimal points within the same visual block or table column
- keep currency labels, signs, and parentheses readable while making amounts scan vertically
- tables should keep `|` columns aligned and preserve tree indentation / selection markers
- keyboard shortcuts after `---` can be aligned locally within that shortcut section
- avoid changing labels, values, routes, or wording during styling-only passes

scoped shortcuts
- global actions are canonical
- scoped shortcuts should reuse canonical action ids/forms
- scoped routes pass context/defaults into canonical actions
- menus should render from action ids so labels/order/key numbers do not drift
- scoped list views are canonical list views with filters applied
- scoped create flows are canonical create forms with pre-filled fields
- pre-filled fields remain editable unless explicitly locked
- labels can be context-aware, but label logic should live with the canonical action

resource route shape
- resource routes are list-first, not branch menus
- list/history browsing lives under `/list/`, and parent resource actions open those lists directly
- lists own browsing, filtering, and list-scoped shortcuts
- append/create forms live under explicit action routes like `/add/`, `/create/`, or domain verbs
- after successful add/edit/delete, return to the relevant `/list/` page when that page confirms the result
- use `create` for new containers/objects and `add`/`allocate` for appending records/events to existing objects
- ctrl+n opens the matching create/add flow from lists
- ctrl+h cycles hidden visibility on lists that support hidden resources
- examples:
    - `/` accounts opens `/accounts/list/`
    - `/accounts/{account}/` balances opens `/accounts/{account}/balances/list/`
    - `/budgets/{budget}/allocations/list/` uses ctrl+n or a domain shortcut to allocate
    - `/owed/ledgers/{ledger}/transactions/list/` uses ctrl+n to add an owed transaction

session action history / undo support
- everytime a mutation occurs (create account / edit something), we log it above
- this way, when Ctrl-C and exit, its easily searchable (eg. via tmux) previous actions
- also super clear what Ctrl-Z does, it really just undoes the previous action
- visible session history behaves like an undo stack
- visible session history only contains undoable mutations from the current session
- persisted history behaves like effective mutation history: a single-branch recovery log for the current database state
- current-session undo history and persisted effective history can share the same action/mutation schema, but should not behave the same in the UI
- this also means this needs to be a first class citizen, baked deep into the architecture
- literally any mutation, needs a way to undo, and this needs to be backed by compile time checking of interface, and also sufficient unit testing coverage to ensure correctness
- service-level mutations should go through a shared mutation/history boundary so history and undo behavior are not optional per screen
- models should call services, not repos/db directly, so UI paths cannot bypass mutation history
- what this unlocks is efficiency gains. not afraid to do things fast because, u can easily edit or undo. 
- keeps things "simple" as well, we can skip confirmation pages for a lot of otherwise seemingly destructive actions
- persisted history is not an audit log
- it describes mutations currently in effect, not every action ever attempted
- on undo, the corresponding history row is silently deleted, not marked or appended to
- this is intentional: history must reflect what is actually in effect, not what was ever done
- this keeps history aligned to the current effective branch and avoids confusing stale trails
- future undo-via-history is still possible because the JSON blob contains enough info to reconstruct reversals

backups
- its really just all about copying the sqlite
- for now, for simplicity, we no need WAL, cuz its just one user, this also keeps backups simple, can scale later on in the future if needed

database startup
- active database file is db.sqlite in current working directory for v1
- if db.sqlite does not exist, create it and run migrations
- if db.sqlite exists, verify it is sqlite
- verify it is a stuf database using app metadata/migration table
- run pending migrations every startup
- after migrations, validate required schema
- if validation fails, stop with clear error

## user journey

### starting from scratch

goals 
- ux should guide users into inputting data naturally

journey
- user opens app
- on init app
    - look for db in current dir
        - if none, create one, and seed default currencies conversion rates (relative to USD)
        - if have,
            - run migrations (if any)
    - look for config file (empty counts too, eg. current dir)
    - if none, 
        - create global config file
        - add comment which links to github repo for config docs
        - try to init app currency based on current location
        - if location detection fails, default app currency to USD and warn user
    - if have, 
        - validate
        - invalid config stops app startup with a clear error
        - recovery path is to fix or delete the config file, then relaunch
- user should be greeted with a dashboard which then shows different information, and action choices
- the dashboard information should hint at what the users need to input, and users can easily see with the actions at the bottom
- below is a quick draft
- total would be 0, total of on-budget accounts, user would question it, then see the first action to be accounts

account flow decisions
- fresh dashboard shows real empty values, not demo data
- account balance usually means the latest balance entry
- parent account display balance can be derived from child account balances when the parent has no own balance
- if no balance has been added and no child balance can derive it, balance is shown as 0
- creating an account does not ask for an opening balance
- after creating an account, redirect to /accounts/list/
- mutation history is enough success feedback
- esc means back everywhere except /, where it opens exit confirmation
- left/h and right/l follow the horizontal navigation rules above
- on menus, left/h and esc both go back; right/l and enter both open/confirm
- on filterable lists, h/l type into the filter; left/right go back/open
- on lists with a create/add flow, ctrl+n opens the matching new element form
- on lists with edit/delete flows, ctrl+e opens edit and ctrl+d deletes the selected element
- forms opened from a list return to that list after success and reselect the edited item when visible
- on forms, ctrl+s submits immediately as if `[confirm]` was focused and enter was pressed
- on list-backed detail screens opened from a list, left/h and right/l move prev/next in that list before menu shortcuts apply
- esc from a create form discards the draft immediately
- ctrl-c quits immediately and gracefully
- quitting clears undo history
- at /, exit confirmation replaces the normal home actions and defaults to no
- only show "undo history will be cleared" in exit confirmation if current session undo history exists
- q is not a shortcut for now
- number hotkeys work only in menu screens
- in forms, numbers are visual labels only
- account names are strict slugs
- fresh app does not seed tags
- tags are shared metadata across accounts and transactions
- account tags inherit down the account tree for filtering and list display
- direct tags are tags assigned directly to a record
- effective account tags are direct account tags plus inherited parent account tags
- account tag inheritance has no per-child exclusion for v1
- if inherited behavior is unwanted, move the tag from the parent to selected children
- child account direct tags are still editable independently from inherited parent tags
- accounts have exactly one currency for v1
- multi-currency institutions can be modeled as parent accounts with child accounts
- balance entries inherit account currency
- account name is a user-facing slug and can change
- internal account id should be immutable
- currencies are system/reference data, not user-created tags
- seed common default currencies for v1
- custom currency creation is not supported yet

account trees
- account trees are similar in spirit to transaction trees: children explain part or all of a parent without double counting the parent and children together
- accounts can optionally have a parent account
- child accounts are normal accounts with their own currency, balances, transactions, and notes
- child accounts inherit the parent account's on-budget status
- parent and child on-budget status must match
- parent accounts may have their own balance entries, but do not need them
- child balances explain part or all of the parent balance
- parent remaining = parent balance - converted child balances
- if a parent account has no own balance, display the converted child total as the parent balance and show remaining as 0
- account totals and reports count child accounts plus parent remaining
- account totals and reports never count parent balance plus child balances together
- remaining rows are virtual/read-only rows
- moving child accounts between parents is deferred for v1

language
- create = make a new object/container
- add = append a value/event/record to an existing object
- create account
- add balance
- edit balance
- delete balance

history format
- {date} {time} {verb} {path}
- history is shown oldest at the top, newest at the bottom
- 2026-05-17 17:30 create /accounts/hsbc-one
- 2026-05-17 17:35 add /accounts/hsbc-one/balances/2026-05-21
- 2026-05-17 17:40 edit /accounts/hsbc-one/balances/2026-05-21
- 2026-05-17 17:45 delete /accounts/hsbc-one

money storage
- do not store money as floats
- store amount as integer + scale, even if v1 only accepts fiat 2-decimal balances
- eg. HKD 50,000.00 -> amount = 5000000, scale = 2
- eg. BTC 0.12345678 -> amount = 12345678, scale = 8

currency conversion
- account currency = balance currency
- transaction currency = event/explanation currency
- app currency = dashboard/report display currency
- parent transaction currency = explanation anchor currency for child totals
- common currencies are seeded from app data
- conversion rates are seeded from app data relative to USD
- seeded currency data lives in the repo
- seeded currency data is embedded in the app binary
- updating the app can update seeded conversion rates
- app startup seeds missing currency data into db.sqlite
- no runtime network fetch is required for currency conversion in v1
- app uses the latest available seeded/cached rate
- missing conversion data should show original currency and omit converted totals with a clear warning
- historical conversion snapshots are deferred for v1
- manual conversion rate override is deferred for v1

currency seed refresh
- currency seed data is generated during development, not fetched at runtime
- `make refresh-currencies` regenerates `internal/seed/currencies.json`
- rates come from the ECB daily euro foreign exchange reference XML
- currency names and decimal precision come from staticdata.dev ISO 4217 metadata
- the generated JSON remains embedded in the app binary for offline startup/seeding

historical conversion rationale
- stuf is balance-anchored, not transaction-ledger-perfect
- latest balances are the truth
- converted totals are present-day approximations for analysis
- old detailed records can be fragmented without corrupting balance-derived growth
- latest seeded/cached rates are enough for v1 current views
- if exact historical fx matters later, add per-transaction/per-owed-transaction rate snapshots

cross-cutting data rules
- notes are plain text
- notes can be multiline
- notes have no markdown semantics for v1
- tags are shared metadata across supported records
- tags use separate join tables per record type, for example account_tags and transaction_tags
- tag names are strict slugs with slash hierarchy support
- tags have immutable internal ids
- tag names are globally unique
- tags can be renamed
- tag rename is a first-class undoable mutation
- tags have notes
- fresh app does not seed tags
- tags are not hidden for v1
- tag hiding is not planned unless real usage shows a need
- tag deletion is deferred for v1
- direct tags are tags assigned directly to a record
- effective account tags are direct account tags plus inherited parent account tags
- account tag inheritance has no per-child exclusion for v1
- transaction tags do not inherit account tags by default
- transactions support income and expense types for v1
- transfers are not supported as transactions for v1
- transaction amount is stored as a non-negative integer plus scale
- transaction type determines direction/sign in reports and summaries
- budgets are intentionally decoupled from transactions
- transactions have no budget link in the v1 data model or UI
- transaction parent is nullable
- any transaction can have a parent transaction
- transaction tree depth is unlimited conceptually
- transaction trees explain larger transactions, but do not update balances
- reports must avoid double counting transaction trees
- if a transaction has children, reports count child transactions plus parent remaining, not parent plus children
- parent amount = converted children total + remaining
- child transaction forms default to parent date/account when useful
- child transaction account must match parent transaction account for v1
- child transaction date stays editable
- parent and child transactions can have different currencies
- child transaction amounts convert into parent transaction currency for explained/remaining math
- common currencies are seeded from app data
- conversion rates are seeded from app data relative to USD
- conversion data lives in the repo and updates with app releases for v1
- runtime network currency refresh is deferred for v1
- manual conversion rate override is deferred for v1
- event/child records can be deleted undoably for v1
- container/master records are not deleted for v1
- accounts and budgets can be hidden
- categories, people, and tags are editable but not hidden/deleted for v1

v1 edge rules before schema
- account currency = balance currency
- transaction currency = event/explanation currency
- app currency = dashboard/report display currency
- parent transaction currency = explanation anchor currency for child totals
- transactions support exactly income and expense types for v1
- transfers are not supported as transactions for v1
- transaction amount is stored as a non-negative integer plus scale
- transaction type determines whether the amount contributes as income or expense
- each transaction has exactly one currency
- transaction currency defaults to selected account currency
- transaction currency is editable in create/edit forms
- amount is entered in transaction currency
- transaction currency can differ from account currency
- default currency/rate data lives in the repo and is embedded in the app binary
- app startup seeds missing currency/rate data into db.sqlite for fast local lookup
- latest seeded/cached rates are used for conversion in v1
- historical conversion snapshots are deferred
- manual conversion rate override is deferred
- historical conversion snapshots are not needed for v1 because balance snapshots anchor truth
- converted totals are present-day approximations for analysis, not exact historical ledgers
- parent and child transactions can have different currencies
- child amounts convert into parent currency for explained/remaining math
- changing parent transaction currency does not change child transaction currencies
- changing parent transaction currency recalculates explained/remaining with latest rates
- parent remaining is calculated across all children regardless of report period
- if conversion is missing, show the original currency and omit affected rows from converted totals with a warning
- effective report rows count in the coverage period containing their own transaction date
- parent remaining row counts on the parent transaction date
- child rows can appear in a different report period from their parent
- coverage period determines inclusion, not only calendar month
- mixed-type children are blocked in v1 UI
- expense parents can only have expense children in v1 UI
- income parents can only have income children in v1 UI
- child transaction account must match parent transaction account for v1
- deleting a transaction with children is blocked in v1
- user must delete children before deleting the parent transaction
- tags are shared metadata across accounts and transactions
- notes are the general breadcrumb field across records
- tags are primarily for filtering/querying and reusable meaning
- accounts use account_tags for v1
- transactions use transaction_tags when transactions land
- future taggable records can add their own join tables
- each owed ledger has exactly one viewing/anchor currency
- different owed ledgers can use different currencies
- owed ledger currency defaults to app currency
- each owed transaction has exactly one currency
- owed transaction currency defaults to owed ledger currency
- owed transaction currency is editable in create/edit forms
- owed transaction amount is entered in owed transaction currency
- owed transaction amount converts into owed ledger currency for running balance
- dashboard `ppl owe you` converts ledger balances to app currency
- missing owed transaction conversion blocks confirm
- if no config exists, try location-based app currency
- if location detection fails, use USD
- if USD fallback is used, warn user that app currency defaulted to USD and can be changed in config
- invalid config stops app startup with a clear error
- config recovery path is to fix or delete the config file, then relaunch

dashboard questions
- the dashboard is the quick pulse check
- how am i doing so far?
- is this drop normal?
- how bad was the drawdown?
- where is my floor?
- for now, high-to-low is balance rhythm context, not exact expenses
- later, once income / transactions exist, inferred expenses should sit near high-to-low because one shows what went out and the other shows how far balances fell
- `net changes` means month start to month end for completed months, and month start to dashboard `as-of` for the current month
- `high to lows` means each month's high-to-low drawdown
- `lows` means each month's lowest known / carried on-budget balance floor
- high-to-high is useful, but it is more of a report question about peak capacity
- the dashboard shows lows directly instead of low-to-low trend because the floor is more useful at a glance

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

ppl owe you : HKD       0.00

/

> 1) accounts
  2) transactions
  3) budgets
  4) owed
  5) reports
  6) settings
  7) backup

---
up/down : navigate
left/h  : back
right/l : open
enter   : confirm
esc     : exit app
?       : help
```

- keyboard shortcuts shown are for basic navigation
    - j/k, tab/shift-tab can also navigate vertically
    - left/h and right/l provide yazi-style back/open on menus
    - 1/2/3/4/5/6/7 hotkeys
    - number hotkeys only work in menu screens, not forms

- at /, esc opens exit confirmation
- exit confirmation replaces the normal home actions
- no is selected by default
- esc from exit confirmation cancels and returns to normal /
- ctrl-c quits immediately and gracefully
- quitting clears undo history
- only show "undo history will be cleared" if current session undo history exists

```
# stuf

total       : HKD 0.00
budgeted    : HKD 0.00

period      : 2026-05

growth
on-budget   : HKD 0.00
total       : HKD 0.00

ppl owe you : HKD 0.00

/

exit app?

> 1) no
  2) yes

---
up/down : navigate
enter   : confirm
esc     : cancel
ctrl-c  : quit
?       : help
```

```
history (ctrl-z to undo)
- 2026-05-17 17:30 create /accounts/hsbc-one

# stuf

total       : HKD 0.00
budgeted    : HKD 0.00

period      : 2026-05

growth
on-budget   : HKD 0.00
total       : HKD 0.00

ppl owe you : HKD 0.00

/

exit app?
undo history will be cleared

> 1) no
  2) yes

---
up/down : navigate
enter   : confirm
esc     : cancel
ctrl-c  : quit
?       : help
```

- pressing ? shows
    - short description of each action
    - the help should change based on the current context
    - other hidden keyboard shortcuts
    - press ? again, or esc to exit help

- user presses 1 (accounts)
- dashboard still shows, cuz thats the context in which the user decided to select accounts for
- esc becomes back instead of quit

```
# stuf

total       : HKD 550,000.00
budgeted    : HKD   3,000.00

period      : 2026-05

growth
on-budget   : HKD   5,200.00
total       : HKD  36,200.00

ppl owe you : HKD     456.00

/accounts/list/

total       : HKD 550,000.00
on-budget   : HKD  50,000.00
off-budget  : HKD 500,000.00

showing     : non-hidden

> filter    : (type anything...)

    on-budget accounts
    name             | balance                         | notes        | tags
    TOTAL            | HKD  50,000.00                  |              |

  > hsbc-one         | HKD  47,400.00                  |              | owner/me, region/hk
      hsbc-hkd       | HKD  35,000.00                  | daily cash   | owner/me, region/hk
      hsbc-usd       | HKD   7,800.00 (USD 1,000.00)   |              | owner/me, region/hk
      hsbc-cad       | HKD   4,600.00 (CAD   800.00)   |              | owner/me, region/hk

    wallet           | HKD   2,600.00                  |              | owner/me

    off-budget accounts
    name             | balance                         | notes        | tags
    TOTAL            | HKD 500,000.00                  |              |

    investment       | HKD 500,000.00                  | broker total | owner/me, kind/investment
      investment-usd | HKD 320,000.00 (USD 41,025.64)  |              | owner/me, kind/investment
      investment-hkd | HKD 100,000.00                  |              | owner/me, kind/investment
      remaining      | HKD  80,000.00                  |              |

---
type          : filter
h/l           : type in filter
up/down       : navigate
left/right    : back/open
enter         : confirm
ctrl+n        : new
ctrl+e        : edit
ctrl+t        : tags
ctrl+h        : hidden
esc           : back
?             : help
```

- user presses ctrl+n to create

- dashboard hides, focus on create account flow
- keyboard shortcuts changes too, as we are now in /accounts/create/
- arrow keys dont navigate, as it conflicts with 
- tab/shift-tab becomes "navigate"
- the input fields change how they are rendered based on focus state

- for name input (this will be a text input component, option: single-line)
- nothing entered will give a placeholder "(type anything...)"
- typing name enforces lowercase, no space, no special char (alphanumeric and hyphens only)
    - implementation-wise... perhaps the text input component have the option of passing in some sort of post-processing logic
- enter will go to next field

```
# stuf

/accounts/create/

> 1) name      : (type anything...)

  2) currency  : HKD

  3) on-budget : true

  4) notes     :

  5) tags      : []

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

- for currency input
- select input component, multi-select = false, can-filter = true, can-create = false, default = app currency, show pagination = true
- account balance entries inherit account currency
- users cannot create currencies from account creation
- currency options come from the currency table
- if a currency is missing, user should update currency data or configure it in settings later

```
# stuf

/accounts/create/

  1) name      : hsbc-one

> 2) currency  : HKD

     > HKD
       USD
       CAD

  3) on-budget : true

  4) notes     :

  5) tags      : []

  [confirm]

---
type       : filter
h/l        : type in filter
up/down    : move cursor
left/right : next/prev page
enter      : confirm
tab        : navigate
esc        : back
?          : help
```

- for on-budget input
- select input component, multi-select = false, can-filter = false, can-create = false, default = "true", show pagination = false
- share component with multi-select cuz we want to share the keybinds and logic, prevent drift

```
# stuf

/accounts/create/

  1) name      : hsbc-one

  2) currency  : HKD

> 3) on-budget : true

     > true
       false

  4) notes     :

  5) tags      : []

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

- notes is also a text input, like name
- options: newline - true
- shift enter can newline
- can be empty, so enter / tab will go next

```
# stuf

/accounts/create/

  1) name      : hsbc-one

  2) currency  : HKD

  3) on-budget : true

> 4) notes     : (type anything...)

  5) tags      : []

  [confirm]

---
type        : enter text
tab         : navigate
enter       : confirm
shift-enter : newline
esc         : back
?           : help
```

- for tags input
- tags input is a select input component
- multi-select = true, can-filter = true, can-create = true, default = []
- account tags use the same shared tag vocabulary as transactions
- if no exact match for the filter exists, show create as the last option
- inline-created tags use the typed slug and empty notes
- add asterisk for new tags until submit
- tags already added should not show up in the tag option list
- account create submits the account, inline-created tags, and account tag assignments as one undoable service-level mutation
- account edit with tags submits account field changes, inline-created tags, and account_tags changes as one undoable service-level mutation
- if any part of account create/edit with tags fails, the whole mutation fails

```
# stuf

/accounts/create/

  1) name      : hsbc-one

  2) currency  : HKD

  3) on-budget : true

  4) notes     :

> 5) tags      : []

   > filter    : owner

     > owner/me
       owner/wife
       (create new "owner")

     [03/03]

  [confirm]

---
type       : filter
h/l        : type in filter
up/down    : move cursor
left/right : next/prev page
enter      : confirm
backspace  : delete last tag
tab        : navigate
esc        : back
?          : help
```

- on the last option "confirm", note the change in keyboard shortcuts
- tab does nothing cuz already at the last, so show shift-tab cuz can go back up

```
# stuf

/accounts/create/

  1) name      : hsbc-one

  2) currency  : HKD

  3) on-budget : true

  4) notes     :

  5) tags      : [owner/me, region/hk]

> [confirm]

---
shift-tab : navigate
enter     : confirm
esc       : back
?         : help
```

- if confirmed failed for some reason, show error but dont crash the app (would be frustrating to re-enter)
- can be anything but likely would be backend validation logic (eg. somehow the name is not all lowercase maybe)
- the principle is that  
    - backend should hv validation in addition to frontend logic
    - should not error silently
    - should not crash if error is recoverable or not fatal
- general error behavior
    - error should remain as long as we are still in this page
    - error should disappear if we quit to the previous page (error no longer relevant)
    - error should disappear after we successfully create account (see below)

```
# stuf

/accounts/create/

  1) name      : hsbc-one-INVALIDCHARS!)(%@*)

  2) currency  : HKD

  3) on-budget : true

  4) notes     :

> [confirm]

  [!] ERROR: NAME - INVALID CHARACTERS DETECTED

---
shift-tab   : navigate
enter       : confirm
esc         : back
?           : help
```

- after confirm success, goes to /accounts/list/ automatically, serves a few purposes
    - quickly confirms that the account has been created successfully
    - user tends to want to do something with that account after it has been created
- accounts list should be filterable
- perhaps can reuse the multi-select component... or multi-select component should be built from reusable components that this can use
- filterable because there can be a LOT of accounts
- plain text account filtering searches account name, notes, effective tag names, currency code, and currency name
- structured account filters can use GitHub-style terms
- space between structured terms means AND
- comma inside one structured term means OR
- bare words search text-like fields
- `tag:me,wife` matches accounts with me OR wife
- `tag:me tag:wife` matches accounts with me AND wife
- `currency:HKD` matches the account currency code
- listed alphabetically by default... think about alternative sorting in the future but, alphabetical works as a good default cuz, can just rename them with number prefixes
- split by on/off budget, but arrow keys and filters should filter both
    - hide either category if no search results for either one
    - if no search results for both, see handling below, (no results)

- here we should also be able to have a birds eye view of account stuff like totals
- accounts list shows a summary above the filter/table with total, on-budget, and off-budget totals
- accounts list summary totals are scoped to the active account filter
- when the filter is empty, summary totals describe all visible accounts in the current hidden/non-hidden mode
- when the filter is active, labels can use filtered total to make the scope clear
- filtered totals preserve account tree no-double-counting rules
- if a matching parent has children, totals count matching children plus matching parent remaining, not parent balance plus children
- parent remaining participates in filtered totals only when the parent account itself is in the filtered scope
- if only a child matches the filter, totals count that child without pulling in the parent remaining
- accounts list shows effective account tags in the rightmost column
- effective account tags include direct tags plus inherited parent tags
- accounts list does not visually distinguish direct and inherited tags
- account detail and edit views can show direct and inherited tags separately
- ctrl+t from /accounts/list/ opens /tags/list/ to manage the shared tag vocabulary

- do note that history is added!
- visible history above is shown for the current session only
- visible history is shown oldest at top, newest at bottom
- visible history behaves like an undo stack
- ctrl-z undoes the latest visible history row
- after undo succeeds, remove that row from visible history
- undo does not add a visible history row
- visible history is cleared when the app exits
- persisted history should still be stored in db, so effective mutations can be inspected or reconstructed by future recovery tooling
- persisted history behaves like effective mutation history: a single-branch recovery log for the current database state
- persisted history survives app restarts
- persisted history stores old/new data for recovery, but v1 does not support ctrl-z for previous-session mutations
- current-session undo history and persisted effective history can use the same action/mutation schema, but should not behave the same in the UI
- since history is stored in db, the db schema can also be much simpler, no need for each table to support soft deletes, as all deletes are soft by default, assuming all actions are undo-able
- after successful undo, return to / and re-render, just to keep things simple for now and prevent any rendering bugs
- the language we go for {date} {time} {verb} {path}, we can update further in the future
- history db should store enough old/new JSON data to deterministically reconstruct or inspect effective mutations when recovery tooling exists
- to keep things simple... store json data, like the create -> old is null, new has json, update -> old has json, new has json, represents the diff, delete -> old has json, new is null
- ctrl-z example
    - before ctrl-z
        - 2026-05-17 17:30 create /accounts/hsbc-one
        - 2026-05-17 17:35 add /accounts/hsbc-one/balances/2026-05-21
        - 2026-05-17 17:45 delete /accounts/hsbc-one/balances/2026-05-21
    - after ctrl-z
        - 2026-05-17 17:30 create /accounts/hsbc-one
        - 2026-05-17 17:35 add /accounts/hsbc-one/balances/2026-05-21

```
history (ctrl-z to undo)
- 2026-05-17 17:30 create /accounts/hsbc-one

# stuf

total       : HKD 550,000.00
budgeted    : HKD   3,000.00

period      : 2026-05

growth
on-budget   : HKD   5,200.00
total       : HKD  36,200.00

ppl owe you : HKD     456.00

/accounts/list/

total       : HKD 550,000.00
on-budget   : HKD  50,000.00
off-budget  : HKD 500,000.00

showing     : non-hidden

> filter    : (type anything...)

    on-budget accounts
    name             | balance                         | notes        | tags
    TOTAL            | HKD  50,000.00                  |              |

  > hsbc-one         | HKD  47,400.00                  |              | owner/me, region/hk
      hsbc-hkd       | HKD  35,000.00                  | daily cash   | owner/me, region/hk
      hsbc-usd       | HKD   7,800.00 (USD 1,000.00)   |              | owner/me, region/hk
      hsbc-cad       | HKD   4,600.00 (CAD   800.00)   |              | owner/me, region/hk

    wallet           | HKD   2,600.00                  |              | owner/me

    off-budget accounts
    name             | balance                         | notes        | tags
    TOTAL            | HKD 500,000.00                  |              |

    investment       | HKD 500,000.00                  | broker total | owner/me, kind/investment
      investment-usd | HKD 320,000.00 (USD 41,025.64)  |              | owner/me, kind/investment
      investment-hkd | HKD 100,000.00                  |              | owner/me, kind/investment
      remaining      | HKD  80,000.00                  |              |

---
type          : filter
h/l           : type in filter
up/down       : navigate
left/right    : back/open
enter         : confirm
ctrl+n        : new
ctrl+h        : hidden
ctrl+t        : tags
esc           : back
?             : help
```

- account balance usually shows the latest added balance
- parent account balance can be derived from child account balances when the parent has no own balance
- if the account has no balances and no child balance can derive it, the balance is shown as 0
- accounts list shows app currency first for comparison
- if account currency differs from app currency, show account currency in parentheses
- pressing enter on an account opens the account detail page
- ctrl+e edits the selected account directly from the list
- ctrl+d is reserved for lists with delete flows
- empty accounts can be deleted undoably
- non-empty accounts should be hidden instead of deleted

```
history (ctrl-z to undo)
- 2026-05-17 17:30 create /accounts/hsbc-one

# stuf

name        : hsbc-one
balance     : HKD 47,400.00
children    : HKD 47,400.00
remaining   : HKD      0.00
as of       : 2026-05-21
on-budget   : true
tags        : [owner/me, region/hk]
direct tags : [owner/me, region/hk]
inherited   : []
notes       :

/accounts/hsbc-one/

> 1) balances
  2) child accounts
  3) transactions
  4) edit account
  5) hide account

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- hide/show is available for all accounts
- account detail shows effective tags plus direct/inherited tag breakdown
- delete account is shown only when the account is empty
- empty account means no balances and no child accounts
- future transaction support should also make accounts with transactions non-empty
- empty account deletion is immediate and undoable
- non-empty account delete is not shown and should return a friendly error if reached from stale state
- accidental newly-created accounts can be undone with ctrl-z if still the latest history action
- pressing 1 (balances) opens the account balances menu
- pressing 2 (child accounts) opens the account child list
- pressing 3 (transactions) opens an automatically filtered account transactions list
- pressing 4 (edit account) opens the edit account flow
- account transactions is an automatically filtered shortcut to global transactions
- account-scoped transaction list is the global transaction list filtered by account
- account-scoped transaction creation reuses global transaction forms
- account-scoped transaction forms pre-fill account
- pre-filled account remains editable
- only show hidden field if true
- hidden accounts are excluded from the default account list
- hidden accounts preserve balances, transactions, history, and reports where relevant
- hidden accounts can be shown/unhidden from hidden account detail
- user-facing language should say balance, not snapshot
- internally, these may still be implemented as balance snapshots

```
history (ctrl-z to undo)
- 2026-05-17 17:30 create /accounts/investment

# stuf

name        : investment
balance     : HKD 500,000.00
children    : HKD 420,000.00
remaining   : HKD  80,000.00
as of       : 2026-05-21
on-budget   : false
notes       : broker total

/accounts/investment/

> 1) balances
  2) child accounts
  3) transactions
  4) edit account
  5) hide account

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- parent account detail always uses the same fields and math
- if the parent account has no own balance, balance comes from converted child balances and remaining is 0
- if the parent account has its own balance, remaining is parent balance minus converted child balances
- as of uses the latest relevant balance date
- for a parent account with no own balance, as of uses the latest child balance date

```
# stuf

parent      : investment
balance     : HKD 500,000.00
children    : HKD 420,000.00
remaining   : HKD  80,000.00
as of       : 2026-05-21

/accounts/investment/children/list/

> filter : (type anything...)

  name             | balance                         | notes
> investment-usd   | HKD 320,000.00 (USD 41,025.64)  |
  investment-hkd   | HKD 100,000.00                  |

---
type          : filter
h/l           : type in filter
up/down       : navigate
tab/shift-tab : navigate
left/right    : back/open
backspace     : edit filter
enter         : confirm
ctrl+n        : new
ctrl+e        : edit
esc           : back
?             : help
```

- child account lists show parent summary above the URL
- child account lists show child accounts only
- child account lists are filterable by name and notes
- remaining is already shown in the parent summary and does not appear in the child table
- ctrl+n from a child account list opens the child account create form

```
# stuf

parent         : investment
on-budget      : false

/accounts/investment/children/create/

> 1) name      : investment-usd

  2) currency  : USD

  3) notes     :

  4) tags      : []

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

- child account creation uses the normal account creation behavior with parent context
- child account on-budget status is inherited from the parent account
- inherited on-budget status is shown in context, not as an editable form field
- inherited tags are shown in context/detail, but only direct child tags are edited in the child form
- after child account create success, goes to /accounts/{parent}/children/list/ automatically

```
# stuf

/accounts/hsbc-one/transactions/list/

  date       | amount | notes

---
up/down : navigate
enter   : confirm
ctrl+n  : new
enter   : confirm
esc     : back
?       : help
```

```
# stuf

/accounts/hsbc-one/transactions/list/

> filter : (type anything...)

  date       | type    | amount         | notes
> 2026-05-15 | income  | HKD 20,000.00  | salary
  2026-05-16 | expense | HKD    200.00  | groceries

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

```
# stuf

/accounts/hsbc-one/transactions/add-expense/

> 1) date    : 2026-05-21

  2) amount  : (type amount...)

  3) currency: HKD

  4) account : hsbc-one

  5) tags    : []

  6) notes   :

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

- edit account reuses the create account form/input components
- edit account is pre-filled with existing account data
- account name is required
- account name must remain unique
- duplicate account name is rejected
- keeping the same name while editing is allowed
- account parent is not editable for v1
- root account edit shows on-budget
- child account edit does not show on-budget because child account on-budget status is inherited from the parent
- account currency can be edited only if the account has no balances
- if balances exist, currency field is read-only/disabled
- changing currency after balances exist should be modeled by creating a separate account
- after edit success from account detail, goes to the updated account detail page
- after ctrl+e edit success from /accounts/list/, follows the general list-origin rule and returns to /accounts/list/
- if account name changed from detail edit, goes to the new account URL

```
history (ctrl-z to undo)
- 2026-05-17 17:30 create /accounts/hsbc-one

# stuf

/accounts/hsbc-one/edit/

> 1) name      : hsbc-one

  2) currency  : HKD

  3) on-budget : true

  4) notes     :

  5) tags      : [kind/debit-card]

  inherited    : [owner/me, region/hk]

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

- if account has balances, currency is shown but cannot be changed

```
history (ctrl-z to undo)
- 2026-05-17 17:30 create /accounts/hsbc-one
- 2026-05-17 17:35 add /accounts/hsbc-one/balances/2026-05-21

# stuf

/accounts/hsbc-one/edit/

> 1) name      : hsbc-one

  2) currency  : HKD (locked because balances exist)

  3) on-budget : true

  4) notes     :

  5) tags      : [kind/debit-card]

  inherited    : [owner/me, region/hk]

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

```
history (ctrl-z to undo)
- 2026-05-17 17:30 create /accounts/hsbc-one
- 2026-05-17 17:40 edit /accounts/hsbc-main

# stuf

name        : hsbc-main
balance     : HKD 0.00
as of       : (no balance entered yet)
on-budget   : true
tags        : []
direct tags : []
inherited   : []
notes       :

/accounts/hsbc-main/

> 1) balances
  2) transactions
  3) edit account
  4) hide account

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

```
history (ctrl-z to undo)
- 2026-05-17 17:30 create /accounts/hsbc-one

# stuf

name        : hsbc-one
balance     : HKD 0.00
as of       : (no balance entered yet)

/accounts/hsbc-one/balances/list/

  date       | balance | notes
  (no balances yet)

---
up/down/j/k : navigate
left/right  : back/open
enter       : confirm
ctrl+n      : new
ctrl+e      : edit
ctrl+d      : delete
esc         : back
?           : help
```

- pressing balances from account detail opens the account balances list
- ctrl+n from the balances list opens the add balance flow
- date defaults to today
- date is required
- balance is required
- fiat balances accept up to 2 decimal places for v1
- positive, zero, and negative balances are allowed
- balances sort newest first
- stuf intentionally allows only one balance snapshot per account per date
- dates are day-level anchors; v1 does not track time-of-day differences inside a date
- per-day timing is intentionally out of scope because it adds precision without useful clarity for this app
- be intentional about which balance represents that day
- duplicate account/date balances are rejected
- user should edit the existing balance instead of replacing through add

```
history (ctrl-z to undo)
- 2026-05-17 17:30 create /accounts/hsbc-one

# stuf

name         : hsbc-one
balance      : HKD 0.00
as of        : (no balance entered yet)

/accounts/hsbc-one/balances/add/

> 1) date    : 2026-05-21

  2) balance : (type amount...)

  3) notes   :

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

- after confirm success, goes to /accounts/hsbc-one/balances/list/ automatically
- this confirms that the balance has been added successfully
- this also makes it fast to add multiple historical balances

```
history (ctrl-z to undo)
- 2026-05-17 17:30 create /accounts/hsbc-one
- 2026-05-17 17:35 add /accounts/hsbc-one/balances/2026-05-21

# stuf

name        : hsbc-one
balance     : HKD 50,000.00
as of       : 2026-05-21

/accounts/hsbc-one/balances/list/

  date       | balance       | notes
> 2026-05-21 | HKD 50,000.00 | initial balance

---
up/down/j/k : navigate
left/right  : back/open
enter       : confirm
ctrl+n      : new
ctrl+e      : edit
ctrl+d      : delete
esc         : back
?           : help
```

- pressing enter on a balance opens the balance detail page
- ctrl+e edits the selected balance directly from the list and returns to /accounts/hsbc-one/balances/list/ after success
- ctrl+d deletes the selected balance directly from the list
- detail pages show the selected resource, not necessarily the parent summary
- parent identity can be shown as a field when useful
- left/right move between older/newer balances
- only show available left/right shortcuts

```
history (ctrl-z to undo)
- 2026-05-17 17:30 create /accounts/hsbc-one
- 2026-05-17 17:35 add /accounts/hsbc-one/balances/2026-05-21

# stuf

account     : hsbc-one
date        : 2026-05-21
balance     : HKD 50,000.00
notes       : initial balance

/accounts/hsbc-one/balances/2026-05-21/

> 1) edit balance
  2) delete balance

---
up/down/j/k : navigate
left/h      : older
right/l     : newer
enter       : confirm
esc         : back
?           : help
```

- edit balance reuses the add balance form/input components
- edit balance is pre-filled with existing balance data
- keeping the same date is allowed
- changing to another existing date for the same account is rejected

```
history (ctrl-z to undo)
- 2026-05-17 17:30 create /accounts/hsbc-one
- 2026-05-17 17:35 add /accounts/hsbc-one/balances/2026-05-21

# stuf

account      : hsbc-one

/accounts/hsbc-one/balances/2026-05-21/edit/

> 1) date    : 2026-05-21

  2) balance : 50000.00

  3) notes   : initial balance

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

- after edit success, goes to /accounts/hsbc-one/balances/list/ automatically
- delete balance happens immediately
- no confirmation screen for delete balance in v1
- after delete, goes to /accounts/hsbc-one/balances/list/ automatically
- ctrl-z undoes the latest visible history row

```
history (ctrl-z to undo)
- 2026-05-17 17:30 create /accounts/hsbc-one
- 2026-05-17 17:35 add /accounts/hsbc-one/balances/2026-05-21
- 2026-05-17 17:45 delete /accounts/hsbc-one/balances/2026-05-21

# stuf

name        : hsbc-one
balance     : HKD 0.00
as of       : (no balance entered yet)

/accounts/hsbc-one/balances/list/

  date       | balance      | notes
  (no balances yet)

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- if confirmed failed because a balance already exists for that account/date, show error but dont crash the app

```
# stuf

name         : hsbc-one
balance      : HKD 50,000.00
as of        : 2026-05-21

/accounts/hsbc-one/balances/add/

  1) date    : 2026-05-21

  2) balance : HKD 50,000.00

  3) notes   : corrected balance

> [confirm]

  [!] ERROR: BALANCE ALREADY EXISTS FOR 2026-05-21
      edit existing balance instead

---
shift-tab : navigate
enter     : confirm
esc       : back
?         : help
```

- "no results" mockup

```
/accounts/list/

> filter : amex

  (no results)

```

- hidden accounts mockup

```
# stuf

/accounts/list/

showing  : hidden-only

> filter : (type anything...)

  name        | balance      | notes          | tags
> old-account | HKD    0.00  | closed account | owner/me

---
type          : filter
h/l           : type in filter
up/down       : navigate
left/right    : back/open
enter         : confirm
ctrl+n        : new
ctrl+e        : edit
ctrl+h        : hidden
ctrl+t        : tags
esc           : back
?             : help
```

```
# stuf

name      : old-account
balance   : HKD 0.00
as of     : 2026-05-21
on-budget : true
hidden    : true
tags      : [owner/me]
notes     : closed account

/accounts/old-account/

> 1) show account
  2) balances
  3) transactions
  4) edit account

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- deferred for this first slice
    - deleting account
    - budgets
    - preserving dirty create drafts after esc


### tags

- tags are shared breadcrumbs for accounts and transactions
- notes are the general breadcrumb field across records
- tags are primarily for filtering/querying and reusable meaning
- fresh app does not seed tags
- tag names are strict slugs with slash hierarchy support
- tag names are globally unique
- tags have immutable internal ids
- renaming a tag updates displays because records link to the immutable tag id
- renaming a tag records history and can be undone in the current session
- tags have plain-text multiline notes
- tags are not hidden for v1
- tag hiding is not planned unless real usage shows a need
- tags are expected to grow in quantity
- filtering/search/sort/querying are the intended way to manage tag volume
- accounts and budgets can be hidden because their lists are small and stale entries create significant noise
- tags are high-volume metadata, so hiding them would add lifecycle complexity without clear value
- tag deletion is deferred for v1
- account_tags is used for account tag assignment
- transaction_tags is used for transaction tag assignment when transactions land
- future taggable records can add their own join tables
- tag management is not shown on the dashboard for v1
- tag routes are still direct URL/path targets
- ctrl+t from /accounts/list/ opens /tags/list/
- account filters match direct and inherited parent tags
- transaction filters use directly assigned transaction tags
- transaction tags do not inherit account tags by default
- account create/edit with tags is one undoable service-level mutation
- inline-created tags, record changes, and join-table changes succeed or fail together

```
# stuf

/tags/list/

showing  : all

> filter : (type anything...)

  name | notes

---
type          : filter
up/down       : navigate
left/right    : back/open
enter         : confirm
ctrl+n        : new
esc           : back
?             : help
```

```
# stuf

/tags/list/

> filter : (type anything...)

  name        | notes
> bank        | bank-related records
  recurring   | repeated records
  travel      | travel breadcrumbs

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- pressing enter on a tag opens the tag detail page
- tag list sorts alphabetically by default
- tag sort options can come later if needed

```
# stuf

/tags/create/

> 1) name  : (type slug...)

  2) notes :

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

```
# stuf

name  : bank
notes : bank-related records

/tags/bank/

> 1) edit tag

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

```
# stuf

/tags/bank/edit/

> 1) name  : bank

  2) notes : bank-related records

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

tag validation
- name is required
- name must be a strict slug
- name can include slash hierarchy, for example owner/me or kind/credit-card
- name cannot start with slash, end with slash, or contain empty slash segments
- name must be globally unique
- notes are optional

deferred tags
- tag deletion
- tag merge
- tag usage counts
- tag detail backlinks to tagged records


### budgets

- budgets are global envelope-style allocations
- budgets are not monthly category budgets
- budgets carry over by default
- budgets give every dollar a job
- budgets behave like proxy accounts for on-budget money
- creating a budget is separate from allocating money to it
- budgeted = sum of budget balances converted to app currency
- available = on-budget balance - budgeted
- available can be negative
- negative available means money has been spent or allocated beyond current on-budget money
- `ppl owe you` does not increase available
- owed money only becomes real when it appears in on-budget account balances
- budget names are strict slugs
- budget names are globally unique
- budgets have exactly one currency
- budget currency follows account-like rules
- budget currency is fixed once allocations or linked transactions exist
- budget list currency display follows account-list rules
- budget detail does not show a separate currency field because money prefixes imply it
- every budget belongs to exactly one category
- budget categories are grouping labels, not envelopes/proxy accounts
- budget categories are not allocatable
- budget categories are exactly one level deep
- budget categories use strict slugs
- budget categories are globally unique
- categories are user-created
- categories can exist without budgets
- categories are not hidden for v1
- seed built-in category `uncategorized` in the budget schema migration
- `uncategorized` cannot be deleted or renamed for v1
- newly-created budgets default to `uncategorized`
- normal categories are shown even when empty
- `uncategorized` is hidden when empty
- if category deletion is supported later, budgets in that category move to `uncategorized`

hide lifecycle
- accounts and budgets can be hidden
- hidden items are excluded from default lists
- hidden items preserve history and reports where relevant
- hidden items can be shown/unhidden from hidden menus
- deletion is deferred for v1

- home budgets opens `/budgets/list/` directly, matching accounts and account balances
- budget list is grouped by category
- budget list follows accounts-list currency display rules
- `uncategorized` section is omitted when empty

```
# stuf

on-budget : HKD 50,000.00
budgeted  : HKD  3,000.00
available : HKD 47,000.00

/budgets/list/

> filter : (type anything...)

  daily
  name      | balance       | notes
> groceries | HKD 1,000.00  | daily food

  travel
  name       | balance                    | notes
  japan-trip | HKD 5,000.00 (JPY 100,000) |

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

budget categories
- category fields are name and notes
- categories can be created inline from budget create/edit
- category editing is supported from /budgets/categories/
- category hiding is deferred
- category deletion is deferred

```
# stuf

/budgets/categories/list/

> filter : (type anything...)

  name | notes

---
type          : filter
up/down       : navigate
left/right    : back/open
enter         : confirm
ctrl+n        : new
esc           : back
?             : help
```

```
# stuf

/budgets/categories/list/

> filter : (type anything...)

  name          | budgets | notes
> daily         | 2       | recurring day-to-day
  travel        | 1       | trips
  future        | 0       | longer-term savings

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- if `uncategorized` has budgets, show it in category lists
- if `uncategorized` has no budgets, omit it from category lists

```
  name          | budgets | notes
> uncategorized | 1       | default category
  daily         | 2       | recurring day-to-day
  travel        | 1       | trips
  future        | 0       | longer-term savings
```

```
# stuf

name    : daily
budgets : 2
notes   : recurring day-to-day

/budgets/categories/daily/

> 1) budgets
  2) create budget in category
  3) edit category

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- global budget creation is canonical
- category-scoped budget creation is a convenience shortcut
- category-scoped forms pre-fill category
- pre-filled category remains editable
- both flows write to the same budget table

```
# stuf

/budgets/categories/create/

> 1) name  : (type anything...)

  2) notes :

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

```
# stuf

/budgets/categories/daily/create-budget/

> 1) name     : (type anything...)

  2) currency : HKD

  3) category : daily

  4) notes    :

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

- edit category is pre-filled with existing category data
- category name is required
- category name must remain unique
- duplicate category name is rejected
- keeping the same name while editing is allowed
- after edit success, goes to the updated category detail page
- `uncategorized` cannot be edited

```
# stuf

/budgets/categories/daily/edit/

> 1) name  : daily

  2) notes : recurring day-to-day

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

```
# stuf

/budgets/create/

> 1) name                   : (type anything...)

  2) currency               : HKD

  3) category               : uncategorized

  4) has default allocation : false

  5) has goal               : false

  6) notes                  :

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

- currency cannot be created inline
- currency options come from the currency table
- category can be created inline
- create budget and edit budget share the same form/input components
- allocation itself is still separate from budget creation
- edit budget is pre-filled with existing budget data
- currency is locked if allocations or linked transactions exist
- default allocations and goals are deferred for the first budget slice
- future budget create/edit can configure optional default allocation and saving goal
- setting default allocation does not create an allocation
- setting a saving goal does not create an allocation
- has default allocation controls whether default allocation fields are shown
- turning has default allocation from true to false removes the default allocation on confirm
- has goal controls whether goal fields are shown
- turning has goal from true to false removes the goal on confirm
- has default allocation true requires default allocation monthly
- has goal true requires goal target amount and goal target month
- optional dependent fields are hidden when their toggle is false
- apply default allocation is shown only when has default allocation is true

future budget planning create/edit mockups:

```
# stuf

/budgets/create/

  1) name                       : groceries

  2) currency                   : HKD

  3) category                   : daily

> 4) has default allocation     : true

  5) default allocation monthly : HKD 200.00

  6) has goal                   : false

  7) notes                      : supermarket spending

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

```
# stuf

/budgets/create/

  1) name                       : japan-trip

  2) currency                   : JPY

  3) category                   : travel

  4) has default allocation     : true

  5) default allocation monthly : JPY 10,000

> 6) has goal                   : true

  7) goal target amount         : JPY 300,000

  8) goal target month          : 2026-12

  9) notes                      : japan trip

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

```
# stuf

/budgets/groceries/edit/

> 1) name                   : groceries

  2) currency               : HKD

  3) category               : daily

  4) has default allocation : false

  5) has goal               : false

  6) notes                  : supermarket spending

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

```
# stuf

/budgets/groceries/edit/

  1) name                       : groceries

  2) currency                   : HKD

  3) category                   : daily

> 4) has default allocation     : true

  5) default allocation monthly : HKD 200.00

  6) has goal                   : false

  7) notes                      : supermarket spending

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

```
# stuf

/budgets/japan-trip/edit/

  1) name                       : japan-trip

  2) currency                   : JPY

  3) category                   : travel

  4) has default allocation     : true

  5) default allocation monthly : JPY 10,000

> 6) has goal                   : true

  7) goal target amount         : JPY 300,000

  8) goal target month          : 2026-12

  9) notes                      : japan trip

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

```
# stuf

name      : groceries
category  : daily
allocated : HKD 1,000.00
spent     : HKD   200.00
balance   : HKD   800.00
notes     : daily food

/budgets/groceries/

> 1) allocations
  2) transactions
  3) edit budget
  4) hide budget

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- only show hidden field if true

```
# stuf

name      : japan-trip
category  : travel
allocated : HKD 5,000.00 (JPY 100,000)
spent     : HKD 5,000.00 (JPY 100,000)
balance   : HKD     0.00 (JPY       0)
hidden    : true
notes     : completed trip

/budgets/japan-trip/

> 1) show budget
  2) allocations
  3) transactions
  4) edit budget

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

allocations
- allocation is separate from budget creation
- allocation entries are deltas internally
- UI supports set total, add money, remove money
- set total calculates the corresponding delta internally
- set total calculates its delta from the running balance immediately before the new allocation
- allocation date defaults to today
- multiple allocations on the same date are allowed
- allocation running balances sort by date, then created_at, then id
- allocation history should show both change and resulting balance

```
# stuf

/budgets/groceries/allocations/list/

  date       | change       | allocated
> 2026-05-21 | HKD 1,000.00 | HKD 1,000.00

---
up/down : navigate
enter   : confirm
ctrl+n  : allocate
esc     : back
?       : help
```

```
# stuf

/budgets/groceries/allocations/list/

  date       | change        | balance       | notes
> 2026-05-01 | HKD 1,000.00  | HKD 1,000.00  | paycheck
  2026-05-10 | HKD  (200.00) | HKD   800.00  | correction

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

```
# stuf

current     : HKD 800.00

/budgets/groceries/allocations/add/

> 1) action : set total

  2) amount : (type amount...)

  3) date   : 2026-05-21

  4) notes  :

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

- allocation action options are set total, add money, remove money
- after confirm success, goes to /budgets/groceries/allocations/list/ automatically

budget and transaction separation
- budgets are intentionally decoupled from transactions
- transactions do not reduce budget balances
- transactions do not contribute to budget spent for v1
- budget balances move through budget allocations only
- transaction tags can still use the same vocabulary as budgets when useful
- reports may compare budget plans and transaction explanations later, but neither mutates the other

```
# stuf

/budgets/hidden/

> filter : (type anything...)

  name       | category | balance       | notes
> japan-trip | travel   | HKD      0.00 | completed trip

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

budget planning
- saving goals and default allocations are separate concepts
- saving goal = target amount + target month
- default allocation = suggested monthly amount to allocate to a budget
- both build on top of budget allocations rather than making budgets month-bound
- default allocations support easy recurring allocations for monthly expenses
- default allocations can also support recurring allocation toward yearly expenses and saving goals

default allocations
- default allocation is optional
- default allocation is a suggested monthly amount
- default allocation is in budget currency
- default allocation does not auto-allocate money
- default allocation helps future monthly allocation flows
- apply default allocation = creates an allocation using the configured default allocation amount

```
# stuf

name      : groceries
category  : daily
allocated : HKD 200.00
spent     : HKD 150.00
balance   : HKD  50.00
notes     : supermarket spending

default allocation
monthly   : HKD 200.00

/budgets/groceries/

> 1) allocations
  2) transactions
  3) apply default allocation
  4) edit budget
  5) hide budget

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

deferred budgets
- budget deletion
- category deletion
- category hiding
- default allocations and saving goals
- detailed category management beyond create/edit
- recurring/monthly allocation flow
- yearly expense allocation flow
- bulk apply default allocations flow
- budget report drilldowns

### transactions

- transactions are optional explanatory records
- balances remain the source of truth
- transactions do not update account balances
- incomplete or incorrect transactions should not corrupt balance-derived growth
- transaction is familiar user-facing language, keep it
- reports are read-only and consume transaction data
- reports never mutate transaction data
- reports may reveal missing data, but input should happen through explicit input flows
- for v1, prioritize clean input flows over report-to-input shortcuts
- income and expense are explicit transaction types for v1
- amount is always positive
- transaction type determines meaning
- type is implied by add-income/add-expense flows and not shown as an editable field there
- transactions have exactly one currency
- transaction currency defaults to selected account currency
- transaction currency is editable in create/edit forms
- transaction amount is entered in transaction currency
- transaction currency can differ from account currency
- explicit transfer transactions are not supported for v1
- users can often skip transfer entry entirely because balance snapshots anchor growth
- balance snapshots capture the result of transfers
- users do not need to manually input two transactions for a transfer
- fresh balances lazy-reconcile messy transfer details
- global transaction creation is canonical
- account-scoped transaction creation exists as a convenience shortcut
- account-scoped forms pre-fill account
- pre-filled account should still be editable for root transactions
- global and account-scoped flows write to the same transaction table
- account detail exposes transactions as an automatically filtered shortcut to global transactions
- credit card statements/payments are the first parent transaction happy path
- put the parent expense under the credit card account when that account exists
- children explain portions of that parent expense with manually assigned tags
- paying back money owed to others is a normal expense transaction with manually assigned arbitrary tags

transaction trees
- transactions can form parent-child trees
- transaction parent is nullable
- any transaction can have a parent transaction
- transactions without a parent are root transactions
- transaction tree depth is unlimited conceptually
- parent and child transactions are explanatory records
- parent and child transactions do not update balances
- parent amount remains its own amount
- child amounts explain some or all of the parent amount
- child amounts convert into parent currency for explained/remaining math
- parent amount = converted children total + remaining
- if converted children total exceeds parent amount, show negative remaining instead of blocking v1 input
- negative remaining means the explanation currently exceeds the parent amount and should be reviewed
- changing parent transaction currency does not change child transaction currencies
- changing parent transaction currency recalculates explained/remaining with latest rates
- child transaction forms default to parent date/account
- child transaction date remains editable
- child transaction account is locked to the parent transaction account for v1
- child transactions use the same income/expense transaction form components
- mixed-type children are blocked in v1 UI
- expense parents can only have expense children in v1 UI
- income parents can only have income children in v1 UI
- deleting a transaction with children is blocked in v1

transaction tree double counting
- reports count child transactions plus parent remaining, not parent plus children
- if a parent has no children, reports count the parent transaction normally
- if a parent has children, reports count the children and the unexplained parent remaining
- if a child has children, apply the same rule recursively
- parent remaining is calculated across all children regardless of report period
- this allows partial explanation of large transactions without losing the original parent transaction

transaction references
- transactions have an internal immutable database id
- transactions have a user-facing reference id for URLs/history
- transaction refs are sequential and stable
- transaction refs must not be reused after deletes
- transaction refs look like tx-000001
- transaction refs do not encode transaction date, account, type, or amount
- editing transaction fields does not change the transaction ref
- transaction ref is shown in URL/history, not as a detail field

transaction identity
- transactions do not have titles
- users should use tags for reusable meaning
- users should use notes when tags are not enough
- tags are better than titles for querying and cross-cutting analysis
- notes are breadcrumbs, not required names
- transaction detail screens identify transactions by date, amount, account, budget, tags, and notes
- transaction refs are for URLs/history only and should not be shown as content fields

report integration
- income transactions replace assumed income in reports
- if no income transactions exist, income = change `(assumed)`
- if income transactions exist, expenses = income - change `(derived)`
- expense transactions explain derived expenses in reports
- reports consume effective transaction rows, not raw parent + child rows
- reports should not pressure users to enter every expense
- future reports may offer shortcuts to source input flows
- report drilldown should show income/expense explanation without mutating source data
- transaction links/parent-child relationships allow recursive drilldown

```
# stuf

/transactions/list/

  date       | amount | account | notes

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

```
# stuf

/transactions/list/

> filter : (type anything...)

  date       | type    | amount                   | account  | notes
> 2026-05-15 | income  | HKD 20,000.00            | hsbc-one | salary
  2026-04-28 | expense | JPY 12,000 (HKD 620.00)  | hsbc-one | ramen in tokyo
  2026-05-16 | expense | HKD    200.00            | hsbc-one | groceries

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- pressing enter on a transaction opens the transaction detail page
- add income and add expense use the same transaction form/input components
- type is implied by the add flow
- amount is entered as a positive value
- transaction lists show original transaction amount first
- if transaction currency differs from app currency, show app-currency conversion in parentheses
- if conversion is missing, show original currency and a warning marker instead of silently converting
- transaction filters reuse the account filter style where possible
- transaction filters support text, `tag:`, `currency:`, `type:`, negation, comma OR, and repeated-term AND

```
# stuf

/transactions/add-income/

> 1) date    : 2026-05-21

  2) amount  : (type amount...)

  3) currency: HKD

  4) account : hsbc-one

  5) tags    : []

  6) notes   :

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

```
# stuf

/transactions/add-expense/

> 1) date    : 2026-05-21

  2) amount  : (type amount...)

  3) currency: HKD

  4) account : hsbc-one

  5) tags    : []

  6) notes   :

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

- transaction filters reuse the account filter style where possible
- transaction text filters search account, currency, tags, and notes
- `tag:person/alice` matches exact transaction tags
- `currency:HKD` matches transaction currency
- `type:income` and `type:expense` match transaction type
- `-tag:repayment` negates a term
- `tag:food,travel` means OR inside the same term
- repeated terms are AND, for example `type:expense tag:person/alice`

- after add success, goes to /transactions/list/ automatically
- history uses the transaction ref path
- transactions have no budget field in v1 UI

transaction tag input
- tags input is a select input component
- multi-select = true
- can-filter = true
- can-create = true
- default = []
- show pagination = true
- can type anything to filter, fuzzy search
- up/down moves the caret/cursor in the tag option list, not j/k because users need to type
- enter adds the selected tag and does not go to the next field
- tags sort alphabetically by default
- tag sort options can come later, for example last created / last used / most used
- show pagination at the bottom
- use 8 items per page so pagination stays single digit for most lists
- tags do not need number shortcuts because numbers should go into the filter input
- fresh app does not seed tags
- tag options in these mockups are examples of a non-empty tag list
- if no exact match for filter, show create as the last option
- inline-created tags use the typed slug and empty notes
- edit tag later to add breadcrumb notes
- add asterisk for new tags
- if at least one tag exists and the filter is empty, backspace deletes the last tag
- tags already added should not show up in the tag option list
- pagination should update according to the filtered tag list

```
# stuf

/transactions/add-expense/

  1) date    : 2026-05-21

  2) amount  : 200.00

  3) currency: HKD

  4) account : hsbc-one

> 5) tags    : []

   > filter  : (type anything...)

     > credit-card
       groceries
       hkd
       recurring
       supermarket
       travel
       visa
       weekend

     [08/12]

  6) notes   :

  [confirm]

---
type       : filter
h/l        : type in filter
up/down    : move cursor
left/right : next/prev page
enter      : confirm
tab        : navigate
esc        : back
?          : help
```

```
> 5) tags    : []

   > filter  : groc

     > groceries
       (create new "groc")

     [02/02]
```

```
> 5) tags    : [groc*]

   > filter  : (type anything...)

     > credit-card
       groceries
       hkd
       recurring
       supermarket
       travel
       visa
       weekend

     [08/12]

---
type       : filter
h/l        : type in filter
up/down    : move cursor
left/right : next/prev page
enter      : confirm
backspace  : delete last tag
tab        : navigate
esc        : back
?          : help
```

```
> 5) tags    : [groc*]

   > filter  : groceries

     > groceries
       grocery-store

     [02/02]
```

```
> 5) tags    : [groc*, groceries]

   > filter  : (type anything...)

     > credit-card
       hkd
       recurring
       supermarket
       travel
       visa
       weekend

     [07/10]
```

```
history (ctrl-z to undo)
- 2026-05-17 17:35 add /transactions/tx-000001

# stuf

date    : 2026-05-15
type    : income
amount  : HKD 20,000.00
currency: HKD
account : hsbc-one
tags    : []
notes   : salary

/transactions/tx-000001/

> 1) edit transaction
  2) children
  3) add child income
  4) delete transaction

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- edit transaction reuses the add transaction form/input components
- edit transaction is pre-filled with existing transaction data
- transaction type is not editable in v1
- if type is wrong, delete and add the transaction again
- children opens the transaction's child transaction list
- add child income uses the same canonical transaction form with parent defaults
- mixed-type children are blocked in v1 UI

```
# stuf

date      : 2026-05-16
type      : expense
amount    : HKD 10,000.00
currency  : HKD
account   : hsbc-one
children  : HKD  3,500.00
remaining : HKD  6,500.00
tags      : [bank]
notes     : credit card payment

/transactions/tx-000002/

> 1) edit transaction
  2) children
  3) add child expense
  4) delete transaction

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

```
# stuf

larger expense
date      : 2026-05-16
amount    : HKD 10,000.00
currency  : HKD
account   : hsbc-one
tags      : [bank, credit-card]
notes     : credit card payment
explained : HKD  3,500.00
remaining : HKD  6,500.00

/transactions/tx-000002/children/

  date       | type    | amount        | account  | notes
> 2026-05-16 | expense | HKD 1,200.00  | hsbc-one | supermarket
  2026-05-16 | expense | HKD 2,300.00  | hsbc-one | dinner

  1) add child expense

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

```
# stuf

larger expense
date         : 2026-05-16
amount       : HKD 10,000.00
currency     : HKD
account      : hsbc-one
tags         : [bank, credit-card]
notes        : credit card payment
remaining    : HKD  6,500.00

/transactions/tx-000002/children/add-expense/

> 1) date    : 2026-05-16

  2) amount  : (type amount...)

  3) currency: HKD

  4) account : hsbc-one (locked to parent)

  5) tags    : []

  6) notes   :

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

- larger expense context is shown above child lists/forms, not as editable form fields
- remaining is advisory and does not block entry for v1
- child transaction account is locked to the parent transaction account for v1
- child add success goes to /transactions/tx-000002/children/ automatically

```
history (ctrl-z to undo)
- 2026-05-17 17:35 add /transactions/tx-000001

# stuf

/transactions/tx-000001/edit/

> 1) date    : 2026-05-15

  2) amount  : 20000.00

  3) currency: HKD

  4) account : hsbc-one

  5) tags    : []

  6) notes   : salary

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

- delete transaction happens immediately
- no confirmation screen for delete transaction in v1
- delete transaction is blocked if the transaction has children
- user must delete children before deleting the parent transaction
- after delete, goes to /transactions/list/ automatically
- ctrl-z undoes the latest visible history row

transaction validation
- date is required
- amount is required
- amount must be positive
- fiat amounts accept up to 2 decimals for v1
- currency is required
- currency defaults to selected account currency
- transaction currency is editable
- account is required
- child transaction account must match parent transaction account for v1
- tags are optional
- notes are optional

deferred transactions
- report-to-input shortcuts
- parent-child tree visualizations beyond list/detail screens

### reports

- use reports, not reviews
- use change for the first on-budget operating reports
- use growth later when looking at broader assets / off-budget money
- dashboard shows quick balance rhythm for on-budget money
- reports show the period story and account breakdown behind that rhythm
- reports are calendar-period based where applicable
- reports are the place to answer: what happened, how bad did it get, and where did it happen?
- for now, reports are derived from accounts, balances, and transactions only
- as more input flows are added, reports should incorporate budgets, owed/shared money, transactions, tags, and notes
- expect report screens to evolve as those data flows become clearer
- dashboard net change and report change use shared as-of boundary balance rules
- values derived from accounts and balances should be real; only unimplemented domains render as placeholders
- reports are read-only for v1
- reports consume input data but do not mutate it
- balances can be entered on any date
- dynamic values belong in summaries/tables, not option labels
- money decimal points should align
- negative money values should use parentheses
- the first useful report should focus on actionable on-budget accounts
- off-budget / total asset growth can be added later without making the first report noisy
- use income/expenses
- reports only include effective rows whose transaction date is inside the coverage period
- unexplained expenses are coverage-local
- no explained-outside-period bucket for v1
- income comes from income transactions
- before income is entered, income equals change and is marked `(assumed)`
- before income is entered, expenses are 0
- after income is entered, expenses = income - change and can be marked `(derived)`

what questions am i answering
- did my usable on-budget money end higher or lower than it started?
- what was the actual start, end, and change for the month?
- how high did my on-budget money get?
- how low did my on-budget money get?
- how far did it fall from high to low?
- which account caused the change?
- which account caused the scary low / drawdown?
- was this month normal compared to the previous few months?
- which account or month is worth investigating with transactions later?

report metric groups
- balance movement = start, end, change
- liquidity stress = high, low, high-to-low
- keep an empty line between balance movement and liquidity stress
- start/end/change answers where the month ended up
- high/low/high-to-low answers how scary the ride was along the way
- high-to-low is not the same as expenses; it is balance rhythm context
- high-to-high and low-to-low stay useful for longer reports, but the monthly detail should first show the actual high and low values
- later, reports can show high-to-low beside derived expenses because they explain different kinds of monthly load

effective transaction rows
- input screens show original transactions
- reports use effective transaction rows
- if a transaction has no children, it contributes itself as an effective row
- if a transaction has children, it contributes child effective rows plus one parent remaining row when remaining is not 0
- apply the same rule recursively for deeper transaction trees
- effective rows prevent parent + child double counting
- effective rows count in the coverage period containing their own transaction date
- parent remaining row counts on the parent transaction date
- child rows can appear in a different report period from their parent
- coverage period determines inclusion, not only calendar month
- unexplained expenses only compare derived expenses to explained expenses inside the same coverage period
- parent remaining rows are virtual/read-only
- parent remaining rows have no transaction ref
- parent remaining rows keep the parent date/account/type/tags/notes
- report content should not show transaction refs or implementation details
- transaction refs can stay visible in URLs/history only
- opening original transaction from report detail is deferred for v1

expense explanation
- expense explanation order is derived, explained, unexplained
- derived expenses come from balance change and entered income
- explained expenses come from effective expense transaction rows
- unexplained expenses = derived expenses - explained expenses
- unexplained expenses are the remaining expense amount not explained by transactions
- use unexplained, not unknown, because balances are known but details may not be explained yet

```
original transaction tree

2026-05-16 expense HKD 10,000.00 [bank, credit-card] credit card payment
- 2026-05-16 expense HKD 1,200.00 [groceries] supermarket
- 2026-05-16 expense HKD 2,300.00 [restaurants] dinner

effective report rows

2026-05-16 expense HKD 1,200.00 groceries supermarket
2026-05-16 expense HKD 2,300.00 restaurants dinner
2026-05-16 expense HKD 6,500.00 unexplained part of credit card payment
```

report types
- monthly = selected calendar month
- rolling 3 months = latest 3 monthly periods including current report month
- rolling 6 months = latest 6 monthly periods including current report month
- rolling 12 months = latest 12 monthly periods including current report month
- year-to-date = Jan 1 through current/latest period
- annual = selected calendar year, Jan 1 -> Dec 31

```
# stuf

on-budget

current month     : HKD (5,000.00)
rolling 3 months  : HKD  1,200.00
rolling 6 months  : HKD  6,400.00
rolling 12 months : HKD 18,000.00
year-to-date      : HKD 12,400.00

/reports/

> 1) monthly
  2) rolling 3 months
  3) rolling 6 months
  4) rolling 12 months
  5) year-to-date
  6) annual

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- /reports/monthly/ shows a filterable table of monthly summaries
- dynamic values are shown in the table, not in option labels

```
# stuf

current month

period     : 2026-05
coverage   : 2026-05-01 -> 2026-05-31

on-budget
start      : HKD 42,000.00
end        : HKD 37,000.00
change     : HKD (5,000.00)

high       : HKD 45,000.00
low        : HKD 24,000.00
high-to-low : HKD (21,000.00)

/reports/monthly/

> filter   : (type anything...)

  month   | start         | end           | change       | high          | low           | high-to-low
> 2026-05 | HKD 42,000.00 | HKD 37,000.00 | HKD (5,000.00) | HKD 45,000.00 | HKD 24,000.00 | HKD (21,000.00)
  2026-04 | HKD 40,800.00 | HKD 42,000.00 | HKD  1,200.00  | HKD 44,000.00 | HKD 29,000.00 | HKD (15,000.00)

---
type          : filter
h/l           : type in filter
up/down       : navigate
left/right    : back/open
enter         : confirm
esc           : back
?             : help
```

- pressing enter on a month opens the monthly report detail
- monthly report account list is filterable
- monthly report account list starts with on-budget accounts
- off-budget account reporting is deferred until the user explicitly wants the broader asset view
- left/right period navigation is dynamic

```
# stuf

period      : 2026-05
coverage    : 2026-05-01 -> 2026-05-31

on-budget
start       : HKD 42,000.00
end         : HKD 37,000.00
change      : HKD (5,000.00)

high        : HKD 45,000.00
low         : HKD 24,000.00
high-to-low : HKD (21,000.00)

/reports/monthly/2026-05/

> filter   : (type anything...)

  on-budget accounts
  account      | start         | end           | change       | high          | low           | high-to-low
> hsbc-one     | HKD 40,000.00 | HKD 36,200.00 | HKD (3,800.00) | HKD 43,000.00 | HKD 25,000.00 | HKD (18,000.00)
  wallet       | HKD  2,000.00 | HKD    800.00 | HKD (1,200.00) | HKD  2,000.00 | HKD    800.00 | HKD  (1,200.00)
  credit-card  | HKD      0.00 | HKD (8,000.00) | HKD (8,000.00) | HKD      0.00 | HKD (8,000.00) | HKD  (8,000.00)

---
up/down       : navigate
left/h        : previous month
right/l       : next month
enter         : confirm
esc           : back
?             : help
```

- monthly report detail shows balance-derived on-budget movement first
- monthly report account tables render account trees like the account list
- monthly report account tables count child accounts plus parent remaining, not parent plus children
- transaction explanation comes after account movement so the report still starts from balance truth
- expense explanation uses derived, explained, unexplained order

```
# stuf

period      : 2026-05
coverage    : 2026-05-01 -> 2026-05-31

on-budget
start       : HKD 42,000.00
end         : HKD 37,000.00
change      : HKD (5,000.00)

high        : HKD 45,000.00
low         : HKD 24,000.00
high-to-low : HKD (21,000.00)

income
entered     : HKD 20,000.00

expenses
derived     : HKD 25,000.00
explained   : HKD  8,000.00
unexplained : HKD 17,000.00

/reports/monthly/2026-05/expenses/

> filter    : (type anything...)

  date       | amount        | tags          | notes
> 2026-05-16 | HKD 1,200.00  | [groceries]   | supermarket
  2026-05-16 | HKD 2,300.00  | [restaurants] | dinner
  2026-05-16 | HKD 6,500.00  | [bank]        | unexplained part of credit card payment

---
up/down       : navigate
left/h        : previous month
right/l       : next month
enter         : confirm
esc           : back
?             : help
```

- expense explanation rows are effective transaction rows
- remaining rows are virtual/read-only rows
- report expense rows do not show transaction refs or implementation details
- pressing enter on a normal expense row opens the report expense row detail
- pressing enter on an unexplained part opens the remaining expense row detail
- row detail URLs can include transaction refs, but the rendered content should not show them

```
# stuf

date    : 2026-05-16
amount  : HKD 1,200.00
account : hsbc-one
tags    : [groceries]
notes   : supermarket

/reports/monthly/2026-05/expenses/tx-000003/

---
left/h  : previous expense
right/l : next expense
esc     : back
?       : help
```

- normal expense row detail is read-only for v1
- no action list is shown on expense row detail for v1
- left/right navigate to previous/next expense row in the current monthly expense list
- if the expense list is filtered, left/right follow the filtered list order
- hide or disable left/right dynamically when there is no previous/next row
- esc returns to /reports/monthly/2026-05/expenses/

```
# stuf

date      : 2026-05-16
amount    : HKD  6,500.00
account   : hsbc-one
tags      : [bank]
notes     : credit card payment

this is the part of a larger expense that has not been explained yet

larger expense
amount    : HKD 10,000.00
explained : HKD  3,500.00
remaining : HKD  6,500.00

explained by
date       | amount        | tags          | notes
2026-05-16 | HKD 1,200.00  | [groceries]   | supermarket
2026-05-16 | HKD 2,300.00  | [restaurants] | dinner

/reports/monthly/2026-05/expenses/tx-000002/remainder/

---
up/down       : navigate
left/h        : previous expense
right/l       : next expense
esc           : back
?             : help
```

- remaining expense row detail is read-only for v1
- remaining is user-facing language for the parent unexplained part
- the rendered content does not show the parent transaction ref
- the URL uses the parent transaction ref plus /remainder/
- opening original transactions from report detail is deferred for v1

- pressing enter on an account opens the account monthly report detail
- account monthly report detail is the lowest-level account movement detail for now
- no action list is shown at the lowest-level report detail
- only navigation shortcuts are shown
- opening original records from account report detail is deferred

```
# stuf

account   : hsbc-one
on-budget : true
period    : 2026-05
coverage  : 2026-05-01 -> 2026-05-31

start     : HKD 44,800.00
end       : HKD 41,000.00
change    : HKD (3,800.00)

high      : HKD 47,000.00
low       : HKD 29,000.00
high-to-low : HKD (18,000.00)

/reports/monthly/2026-05/accounts/hsbc-one/

---
left/h  : previous month
right/l : next month
esc     : back
?       : help
```

monthly report boundary rules
- monthly periods use shared boundaries, not separate end/start snapshots
- a month start boundary is the first day of that month
- a month end boundary is the first day of the next month
- the end boundary of one month is the same boundary as the start boundary of the next month
- each boundary resolves to the latest balance snapshot on or before that boundary date
- if there is no snapshot on or before the boundary, use the first future snapshot as a flat carried value
- if the boundary is after the latest snapshot, use the latest snapshot as a flat carried value
- if an account has no balances at all, boundary value is 0
- monthly change = resolved end boundary value - resolved start boundary value
- this avoids gaps: April end and May start both use the same resolved value for the May 1 boundary
- example: if snapshots exist on 2026-04-01 and 2026-05-13, the 2026-05-01 boundary uses 2026-04-01
- example: if the first-ever snapshot is 2026-05-02, the 2026-05-01 boundary uses 2026-05-02 as flat history
- monthly high/low candidates are the carried month-start value plus snapshots inside that month

deferred reports
- opening original transactions from report expense detail
- grouped expense views
- rich transaction tree visualizations in reports
- report-to-input shortcuts

### yearly budgeting

- yearly budgeting is handled through saving goals and default allocations later
- a yearly expense is modeled as a budget with target amount and target month
- monthly needed tells the user how much to allocate
- budgets remain global/carry-over, not month-bound
- no separate yearly budget object is needed

### saving goals

- saving goals live under budgets
- saving goal is optional
- one active saving goal per budget
- saving goal currency is budget currency
- saving goals do not make budgets month-bound
- saving goals recommend allocations but do not auto-allocate money
- saving goals are separate from default allocations
- saving goal = where am i trying to get to?
- default allocation = what do i normally put in each month?
- apply default allocation = create an allocation using the configured default allocation amount
- saving goals are configured through edit budget when implemented
- there is no separate goal action/page
- has goal toggles goal fields in edit budget
- turning has goal from true to false removes the goal on confirm
- target amount and target month are required
- target month uses YYYY-MM

goal formulas
- remaining = target amount - budget balance
- months left = number of months through target month
- monthly needed = remaining / months left

```
# stuf

name      : japan-trip
category  : travel
allocated : HKD 5,000.00 (JPY 100,000)
spent     : HKD     0.00 (JPY       0)
balance   : HKD 5,000.00 (JPY 100,000)
notes     : japan trip

goal
target    : JPY 300,000
by        : 2026-12
remaining : JPY 200,000
needed    : JPY  10,527 / month

default allocation
monthly   : JPY 10,000

/budgets/japan-trip/

> 1) allocations
  2) transactions
  3) apply default allocation
  4) edit budget
  5) hide budget

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- goal fields are shown on budget detail when has goal is true
- goal fields are hidden on budget detail when has goal is false
- edit budget is the create/edit/remove flow for saving goals when implemented

deferred saving goals
- goals overview
- multiple active goals per budget
- maintain-balance goals
- automatic recurring allocations
- goal report drilldowns

### investment

- investment tracking is open/deferred
- stuf can track investment account balances as off-budget accounts today
- performance analysis, holdings, cost basis, dividends, and market prices are not v1
- unclear if deep investment tooling belongs inside stuf or as a separate tool
- for v1, investment accounts still contribute to total/off-budget balance snapshots and growth reports

### owed money tracking

- product menu uses `owed`
- dashboard uses `ppl owe you`
- app logic uses owed ledgers
- `owed` tracks money you are owed, not money you owe other people
- money you owe other people should be handled by budgets and normal account/transaction context
- owed ledgers behave like lightweight virtual accounts
- each owed ledger has a currency used as its viewing/anchor currency
- different owed ledgers can use different currencies
- owed ledger currency defaults to app currency
- owed ledger balance is the sum of its owed transactions converted into the ledger currency
- owed ledgers can have positive, zero, or negative balances
- positive ledger balance means unrealized money someone owes you
- negative ledger balance means prepayment, overpayment, or offset
- dashboard `ppl owe you` is the net sum of all owed ledger balances converted to app currency
- `/owed/list/` shows the truth by ledger, even when the dashboard net total is small or zero
- `ppl owe you` does not increase available while open
- owed money only increases available once it appears in on-budget account balances
- owed ledgers do not update account balances
- owed transactions do not update account balances
- owed ledgers are intentionally separate from accounts because they are reminders to collect unrealized money
- real account snapshots remain the source of truth
- if things get messy, enter fresh balances and keep going

owed transactions
- each owed transaction belongs to exactly one owed ledger
- owed transactions are independent records
- each owed transaction has exactly one currency
- owed transaction currency defaults to the owed ledger currency
- owed transaction currency is editable in create/edit forms
- owed transaction amount is entered in owed transaction currency
- positive owed transaction amount means someone owes you more
- negative owed transaction amount means someone paid, prepaid, or offset money
- owed transaction amount converts into owed ledger currency for running balance
- if owed transaction currency differs from owed ledger currency and conversion is missing, confirm is blocked
- owed transactions are not linked to budgets or account transactions for v1
- related transaction links are deferred/context only

owed amount formulas
- owed transaction amount can be a plain signed amount or formula
- formulas start with =
- v1 formulas support numbers, decimals, +, -, *, /, parentheses
- no percentages, variables, functions, or cell refs for v1
- if formula exists, computed amount is used for ledger balances and dashboard totals
- formula is self-documenting input, not separate notes
- formula fields show raw input while focused
- formula fields show computed amount when not focused
- if amount was entered with formula, show formula in parentheses after computed amount when not focused
- if amount was entered manually, show only formatted amount when not focused
- invalid formulas show a recoverable validation error

```
# stuf

ppl owe you : HKD 600.00

/owed/list/

> filter : (type anything...)

  ledger         | balance                    | notes
> alex           | HKD   500.00               | roommate
  ben            | HKD  (200.00)              | prepaid dinner
  cara           | HKD   300.00               | subscriptions
  japan-trip     | HKD 2,150.00 (JPY 40,000)  | trip shares

---
up/down : navigate
enter   : open ledger
ctrl+n  : new ledger
ctrl+e  : edit ledger
esc     : back
?       : help
```

- ledger names are strict slugs
- ledgers can represent humans, groups, trips, subscriptions, or any useful collection target
- ledger names are user-facing and can change
- ledgers have immutable internal ids
- owed transactions link to internal ledger ids
- renaming a ledger updates related owed transaction displays
- pressing enter on a ledger opens ledger detail

```
# stuf

/owed/ledgers/create/

> 1) name     : alex

  2) currency : HKD

  3) notes    : roommate

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : discard
?       : help
```

- after ledger create success, goes to the ledger detail page

```
# stuf

name     : alex
currency : HKD
balance  : HKD 500.00
notes    : roommate

/owed/ledgers/alex/

> 1) transactions
  2) edit ledger

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- pressing 1 (transactions) opens the ledger's owed transaction list

```
# stuf

ledger   : alex
currency : HKD
balance  : HKD 500.00

/owed/ledgers/alex/transactions/list/

  date       | currency | amount       | balance      | notes
  2026-05-01 | HKD      | HKD  500.00  | HKD  500.00 | netflix yearly half
  2026-05-10 | HKD      | HKD (200.00) | HKD  300.00 | partial payback
> 2026-05-28 | HKD      | HKD  200.00  | HKD  500.00 | dinner split

---
up/down : navigate
ctrl+n  : add transaction
ctrl+e  : edit
ctrl+d  : delete
esc     : back
?       : help
```

```
# stuf

ledger   : japan-trip
currency : JPY
balance  : JPY 40,000

/owed/ledgers/japan-trip/transactions/list/

  date       | currency | amount          | balance    | notes
> 2026-05-01 | JPY      | JPY  50,000     | JPY 50,000 | hotel split
  2026-05-03 | HKD      | HKD    (500.00) | JPY 40,000 | paid me back in hkd

---
up/down : navigate
ctrl+n  : add transaction
ctrl+e  : edit
ctrl+d  : delete
esc     : back
?       : help
```

- transaction add only exists from a ledger-scoped transaction list for v1
- `/owed/list/` ctrl+n creates a ledger
- `/owed/ledgers/{ledger}/transactions/list/` ctrl+n adds a transaction for that ledger
- scoped transaction add forms omit the ledger field because the route already supplies it

```
# stuf

ledger  : alex
current : HKD 500.00

/owed/ledgers/alex/transactions/add/

  1) date     : 2026-05-21

  2) currency : HKD

> 3) amount   : =1000/2

  4) notes    :

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

```
  1) date     : 2026-05-21

  2) currency : HKD

  3) amount   : HKD 500.00 (=1000/2)

> 4) notes    : netflix yearly half
```

```
# stuf

ledger  : alex
current : HKD 500.00

/owed/ledgers/alex/transactions/add/

> 1) date     : 2026-05-21

  2) currency : HKD

  3) amount   : (type amount or =formula...)

  4) notes    :

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

- ledger edit is pre-filled with existing ledger data
- ledger name is required
- ledger name must remain unique
- duplicate ledger name is rejected
- keeping the same name while editing is allowed
- ledger currency follows account-like rules
- changing ledger currency recalculates converted running balances with latest rates
- after ledger edit success, goes to the updated ledger detail page

```
history (ctrl-z to undo)
- 2026-05-17 17:40 edit /owed/ledgers/alex-wong

# stuf

/owed/ledgers/alex/edit/

> 1) name     : alex

  2) currency : HKD

  3) notes    : roommate

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

```
# stuf

/owed/ledgers/create/

> 1) name     : (type anything...)

  2) currency : HKD

  3) notes    :

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

- after ledger create success, goes to the ledger detail page

```
# stuf

/owed/list/

> filter : (type anything...)

  ledger         | balance       | notes
> alex           | HKD 500.00    | roommate
  netflix-family | HKD 300.00    | shared subscription

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- after confirm success, goes to the ledger transaction list automatically
- history uses the owed transaction path
- owed transaction refs are sequential, stable, and must not be reused after deletes

```
history (ctrl-z to undo)
- 2026-05-17 17:35 add /owed/ledgers/alex/transactions/txn-000001

# stuf

date     : 2026-05-21
ledger   : alex
currency : HKD
amount   : HKD 500.00
formula  : =1000/2
balance  : HKD 500.00
notes    : netflix yearly half

/owed/ledgers/alex/transactions/txn-000001/

> 1) edit transaction
  2) delete transaction

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- only show formula if amount was entered as formula
- edit transaction reuses the add transaction form/input components
- edit transaction is pre-filled with existing transaction data
- editing formula recomputes amount
- if amount is manually edited, formula is cleared
- after edit success, goes to transaction detail

```
# stuf

/owed/ledgers/alex/transactions/txn-000001/edit/

> 1) date     : 2026-05-21

  2) currency : HKD

  3) amount   : =1000/2

  4) notes    : netflix yearly half

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

- delete transaction happens immediately
- no confirmation screen for delete transaction in v1
- deleting a transaction removes it from the ledger running balance
- after edit/delete success, returns to `/owed/ledgers/{ledger}/transactions/list/`

deferred owed
- related transaction UX
- report integration
- recursive transaction/owed drilldown

### shared finance tracking

- shared finance is mostly a data setup choice for v1
- one db can contain accounts for multiple people if users want shared household visibility
- separate finances can use separate working directories because db.sqlite is local to the current working directory
- truly separate profiles can just be separate folders
- multi-currency is first-class, so separate dbs are not needed just to work around currency limitations
- no first-class owner field is needed for v1
- users can encode ownership in account names and notes if needed
- future account filters/queries can support owner-like views without adding ownership to v1 schema

### settings

- settings are edited through config file for v1
- app currency is the only meaningful setting for now
- date format is fixed ISO and not configurable
- config path is not configurable from the app
- local config in current working directory takes priority if present
- otherwise use global config
- if neither exists, create global config
- new config tries to set app currency from current location
- if location detection fails, app currency defaults to USD
- if USD fallback is used, warn user that app currency can be changed in config
- invalid config stops app startup with a clear error
- config recovery path is to fix or delete the config file, then relaunch
- config files should be safe to delete and regenerate
- in development, use .jsonc as source of truth for defaults, so parsing is always verified and defaults can be embedded in the go binary
- pressing settings shows active config path and app currency

```
# stuf

/settings/

config file : /Users/gjtiquia/.config/stuf/config.jsonc
app currency: HKD

edit settings by editing the config file directly

---
esc : back
?   : help
```

### backup

- active database file is db.sqlite in current working directory for v1
- backup creates timestamped copy of db.sqlite
- backup filename format is db.YYYY-MM-DD-HHMM.sqlite
- no WAL for v1, keeping backups single-file
- backup creates a consistent snapshot and must not race an active write
- restore is manual for v1
- to restore, close stuf and replace db.sqlite with backup file renamed to db.sqlite
- backup does not write undo history
- after backup action, render latest created backup path

```
# stuf

/backup/

database    : /Users/gjtiquia/Documents/self/stuf/db.sqlite
last backup : none

> 1) create backup

restore:
close stuf, replace db.sqlite with your backup, then reopen stuf

---
enter : confirm
esc   : back
?     : help
```

```
# stuf

/backup/

database    : /Users/gjtiquia/Documents/self/stuf/db.sqlite
last backup : /Users/gjtiquia/Documents/self/stuf/db.2026-05-21-1730.sqlite

> 1) create backup

restore:
close stuf, replace db.sqlite with your backup, then reopen stuf

---
enter : confirm
esc   : back
?     : help
```

### export

- export is deferred
- sqlite file is accessible directly for now
- future exports may support csv/json/sqlite

## TUI mockup
