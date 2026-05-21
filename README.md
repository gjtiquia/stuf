# stuf

> ehh... apparently this name is taken... gotta think of a new project name...
> perhaps kuka, kaku, kunga, kwunka, ggaa, gungaa (管家)

```
- [stu]ward [f]inance
- [stuf]f
```

a finance tool

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

things that `stuf` should be able to support
- accounts (on-budget and off-budget)
- multi-currency
- multi-person (can be separate profiles, can be together in one profile, can be "hybrid")
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
- monthly bank statement balances -> net growth/loss
- monthly income -> net cash flow in/out
- lump sum (eg. credit card payment) -> cash flow out sources, percentage of expense, tagging
- transactions -> tagging and deeper analysis; should link to lump sum to prevent "double counting"

lazy reconciliation
- balance snapshots anchor everything
- detailed records can be incomplete without ruining macro analysis
- transactions, budgets, owed items, and settlements explain or plan around balances
- transactions do not update balances
- settlements do not update balances
- budget allocations do not update balances
- if things get messy, enter fresh balances and continue
- the app should feel guilt-free, not like bookkeeping homework

## the implementation 

stack
- golang, bubbletea, sqlite, goose, sqlc

keyboard shortcuts
- separate actions and keys

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

session action history / undo support
- everytime a mutation occurs (create account / edit something), we log it above
- this way, when Ctrl-C and exit, its easily searchable (eg. via tmux) previous actions
- also super clear what Ctrl-Z does, it really just undoes the previous action
- visible session history behaves like an undo stack
- visible session history only contains undoable mutations from the current session
- persisted history behaves like an audit/recovery log
- undo stack and audit log can share the same action/mutation schema, but should not behave the same in the UI
- this also means this needs to be a first class citizen, baked deep into the architecture
- literally any mutation, needs a way to undo, and this needs to be backed by compile time checking of interface, and also sufficient unit testing coverage to ensure correctness
- what this unlocks is efficiency gains. not afraid to do things fast because, u can easily edit or undo. 
- keeps things "simple" as well, we can skip confirmation pages for a lot of otherwise seemingly destructive actions

backups
- its really just all about copying the sqlite
- for now, for simplicity, we no need WAL, cuz its just one user, this also keeps backups simple, can scale later on in the future if needed

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
            - if have network, update cache (eg. currency conversion rates)
    - look for config file (empty counts too, eg. current dir)
    - if none, 
        - create global config file
        - add comment which links to github repo for config docs
        - init currency based on current location (uh if cant... prompt user?)
    - if have, 
        - validate
        - throw error if invalid and irrecoverable
            - eg. no currency will attempt to init currency
            - if currency is not in currency db... (used for unit conversion...) then throw
        - will suggest user to check, or delete
            - implies that, config files should be safe to delete, always
- user should be greeted with a dashboard which then shows different information, and action choices
- the dashboard information should hint at what the users need to input, and users can easily see with the actions at the bottom
- below is a quick draft
- total would be 0, total of on-budget accounts, user would question it, then see the first action to be accounts

account flow decisions
- fresh dashboard shows real empty values, not demo data
- account balance means the latest balance entry
- if no balance has been added, balance is shown as 0
- creating an account does not ask for an opening balance
- after creating an account, redirect to /accounts/list
- mutation history is enough success feedback
- esc means back everywhere except /, where it opens exit confirmation
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
- accounts have exactly one currency for v1
- multi-currency institutions should be modeled as multiple accounts for v1
- grouping related accounts can come later
- balance entries inherit account currency
- account name is a user-facing slug and can change
- internal account id should be immutable
- currencies are system/reference data, not user-created tags
- seed common default currencies for v1
- custom currency creation is not supported yet

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

```
# stuf

total       : HKD 0.00
budgeted    : HKD 0.00

period      : 2026-05

growth
on-budget  : HKD 0.00
total      : HKD 0.00

you owe ppl : HKD 0.00
ppl owe you : HKD 0.00

/

> 1) accounts
  2) transactions
  3) budgets
  4) reports
  5) settings
  6) backup

---
up/down : navigate
enter   : confirm
esc     : exit app
?       : help
```

- keyboard shortcuts shown are for basic navigation
    - j/k, tab/shift-tab can also navigate
    - 1/2/3/4/5/6 hotkeys
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
on-budget  : HKD 0.00
total      : HKD 0.00

you owe ppl : HKD 0.00
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
on-budget  : HKD 0.00
total      : HKD 0.00

you owe ppl : HKD 0.00
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

total       : HKD 50,000.00
budgeted    : HKD  3,000.00

period      : 2026-05

growth
on-budget  : HKD  5,200.00
total      : HKD  6,200.00

you owe ppl : HKD     23.00
ppl owe you : HKD    456.00

/accounts/

> 1) overview
  2) list
  3) hidden
  4) create

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- user presses 3 (create)

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

  4) tags      : []

  5) notes     :

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

  4) tags      : []

  5) notes     :

  [confirm]

---
type       : filter
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

  4) tags      : []

  5) notes     :

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

- for tags input (this will be a select input component, options: multi-select = true, can-filter = true, can-create = true, default = [], show pagination = true)
- can type anything to filter, fuzzy search
- can move arrows up and down which moves the caret/cursor, but not j/k cuz need to type
- enter to add tag, enter does NOT go to next field in this case
- tags sort order... alphabetical by default, can add to config options (eg. last created / last used / most used)
- show pagination at the bottom (8 max cuz... keep it single digit, starts with 1, 9 is ugly, 8 is nicer as a power of two aesthetically)
- tags no need numbering cuz... typing any number should be going into the filter text input anyways, numbers are just unnecessary noise

```
# stuf

/accounts/create/

  1) name      : hsbc-one

  2) currency  : HKD

  3) on-budget : true

> 4) tags      : []

   > filter    : (type anything...)

     > app
       apple
       bank
       cad
       canada
       credit-card
       debit-card
       hkd

     [08/12]

  5) notes     :

  [confirm]

---
type       : filter
up/down    : move cursor
left/right : next/prev page
enter      : confirm
tab        : navigate
esc        : back
?          : help
```

- fresh app does not seed tags
- the above tag options are examples of a non-empty tag list
- if no exact match for filter, show create as the last option

```
> 4) tags      : []

   > filter    : ap

     > app
       apple
       (create new "ap")

     [02/02]
```

- add asterik for new tags *
- if have at least one tag, and nothing is typed in the filter, backspace deletes the last tag (keyboard shortcut only shows backspace if the conditions are met)

```
> 4) tags      : [ap*]

   > filter    : (type anything...)

     > app
       apple
       bank
       cad
       canada
       credit-card
       debit-card
       hkd

     [08/12]

---
type       : filter
up/down    : move cursor
left/right : next/prev page
enter      : confirm
backspace  : delete last tag
tab        : navigate
esc        : back
?          : help
```

```
> 4) tags      : [ap*]

   > filter    : app

     > app
       apple

     [02/02]
```

- tags already added should not show up in the tag list
- note pagination should also update according to the tag list

```
> 4) tags      : [ap*, app]

   > filter    : (type anything...)

     > apple
       bank
       cad
       canada
       credit-card
       debit-card
       hkd
       hong-kong

     [08/11]
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

  4) tags      : [ap*, app, bank, debit-card, hkd, hong-kong]

> 5) notes     : (type anything...)

  [confirm]

---
type        : enter text
tab         : navigate
enter       : confirm
shift-enter : newline
esc         : back
?           : help
```

- on the last option "confirm", note the change in keyboard shortcuts
- tab does nothing cuz already at the last, so show shift-tab cuz can go back up

```
# stuf

/accounts/create/

  1) name      : hsbc-one

  2) currency  : HKD

  3) on-budget : true

  4) tags      : [ap*, app, bank, debit-card, hkd, hong-kong]

  5) notes     :

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

  4) tags      : [ap*, app, bank, debit-card, hkd, hong-kong]

  5) notes     :

> [confirm]

  [!] ERROR: NAME - INVALID CHARACTERS DETECTED

---
shift-tab   : navigate
enter       : confirm
esc         : back
?           : help
```

- after confirm success, goes to /accounts/list automatically, serves a few purposes
    - quickly confirms that the account has been created successfully
    - user tends to want to do something with that account after it has been created
- accounts list should be filterable
- perhaps can reuse the multi-select component... or multi-select component should be built from reusable components that this can use
- filterable because there can be a LOT of accounts
- listed alphabetically by default... think about alternative sorting in the future but, alphabetical works as a good default cuz, can just rename them with number prefixes
- split by on/off budget, but arrow keys and filters should filter both
    - hide either category if no search results for either one
    - if no search results for both, see handling below, (no results)

- here we should also be able to have a birds eye view of account stuff like totals

- do note that history is added!
- visible history above is shown for the current session only
- visible history is shown oldest at top, newest at bottom
- visible history behaves like an undo stack
- ctrl-z undoes the latest visible history row
- after undo succeeds, remove that row from visible history
- undo does not add a visible history row
- visible history is cleared when the app exits
- persisted history should still be stored in db, so that nothing is ever irrecoverable
- persisted history behaves like an audit/recovery log
- persisted history survives app restarts
- persisted history stores old/new data for recovery, but v1 does not support ctrl-z for previous-session mutations
- undo stack and audit log can use the same action/mutation schema, but should not behave the same in the UI
- since history is stored in db, the db schema can also be much simpler, no need for each table to support soft deletes, as all deletes are soft by default, assuming all actions are undo-able
- after successful undo, return to / and re-render, just to keep things simple for now and prevent any rendering bugs
- the language we go for {date} {time} {verb} {path}, we can update further in the future
- history db... should be sufficient to a point such that, even if all the tables are deleted, it can be recoverable via the history db
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

total       : HKD 50,000.00
budgeted    : HKD  3,000.00

period      : 2026-05

growth
on-budget  : HKD  5,200.00
total      : HKD  6,200.00

you owe ppl : HKD     23.00
ppl owe you : HKD    456.00

/accounts/list

> filter : (type anything...)

    on-budget accounts
    name           | balance                         | notes
    TOTAL          | HKD   50,000.00                 |

  > hsbc-one       | HKD   35,000.00                 | main chequing ac
    hsbc-usd       | HKD    7,800.00 (USD 1,000.00) |
    hsbc-cad       | HKD    4,600.00 (CAD   800.00) |

    off-budget accounts
    name           | balance                         | notes
    TOTAL          | HKD  (20,000.00)                |

    investment-hkd | HKD  182,000.00                 |
    student-loan   | HKD (200,000.00)                | negative until fully paid

---
up/down   : navigate
left/right: 
enter     : confirm
esc       : back
?         : help
```

- account balance is the latest added balance
- if the account has no balances yet, the balance is shown as 0
- accounts list shows app currency first for comparison
- if account currency differs from app currency, show account currency in parentheses
- pressing enter on an account opens the account detail page

```
history (ctrl-z to undo)
- 2026-05-17 17:30 create /accounts/hsbc-one

# stuf

name        : hsbc-one
balance     : HKD 0.00
as of       : never
on-budget   : true
tags        : []
notes       :

/accounts/hsbc-one/

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

- account deletion is deferred for v1
- accidental newly-created accounts can be undone with ctrl-z if still the latest history action
- existing accounts should be edited instead of deleted for v1
- pressing 1 (balances) opens the account balances page
- pressing 2 (transactions) opens an automatically filtered account transactions list
- pressing 3 (edit account) opens the edit account flow
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
# stuf

/accounts/hsbc-one/transactions/

> 1) list
  2) add income
  3) add expense

---
up/down : navigate
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

  3) account : hsbc-one

  4) tags    : []

  5) notes   :

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
- account currency can be edited only if the account has no balances
- if balances exist, currency field is read-only/disabled
- changing currency after balances exist should be modeled by creating a separate account
- after edit success, goes to the account detail page
- if account name changed, goes to the new account URL

```
history (ctrl-z to undo)
- 2026-05-17 17:30 create /accounts/hsbc-one

# stuf

/accounts/hsbc-one/edit/

> 1) name      : hsbc-one

  2) currency  : HKD

  3) on-budget : true

  4) tags      : []

  5) notes     :

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

  4) tags      : []

  5) notes     :

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
as of       : never
on-budget   : true
tags        : []
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
as of       : never

/accounts/hsbc-one/balances/

  date       | balance      | notes
  (no balances yet)

> 1) add balance

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- pressing 1 (add balance) opens the add balance flow
- date defaults to today
- date is required
- balance is required
- fiat balances accept up to 2 decimal places for v1
- positive, zero, and negative balances are allowed
- balances sort newest first
- only one balance is allowed per account per date
- duplicate account/date balances are rejected
- user should edit the existing balance instead of replacing through add

```
history (ctrl-z to undo)
- 2026-05-17 17:30 create /accounts/hsbc-one

# stuf

name        : hsbc-one
balance     : HKD 0.00
as of       : never

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

- after confirm success, goes to /accounts/hsbc-one/balances/ automatically
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

/accounts/hsbc-one/balances/

  date       | balance       | notes
> 2026-05-21 | HKD 50,000.00 | initial balance

  1) add balance

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- pressing enter on a balance opens the balance detail page
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
up/down : navigate
left    : older
enter   : confirm
esc     : back
?       : help
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

account     : hsbc-one

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

- after edit success, goes to /accounts/hsbc-one/balances/ automatically
- delete balance happens immediately
- no confirmation screen for delete balance in v1
- after delete, goes to /accounts/hsbc-one/balances/ automatically
- ctrl-z undoes the latest visible history row

```
history (ctrl-z to undo)
- 2026-05-17 17:30 create /accounts/hsbc-one
- 2026-05-17 17:35 add /accounts/hsbc-one/balances/2026-05-21
- 2026-05-17 17:45 delete /accounts/hsbc-one/balances/2026-05-21

# stuf

name        : hsbc-one
balance     : HKD 0.00
as of       : never

/accounts/hsbc-one/balances/

  date       | balance      | notes
  (no balances yet)

> 1) add balance

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- if confirmed failed because a balance already exists for that account/date, show error but dont crash the app

```
# stuf

name        : hsbc-one
balance     : HKD 50,000.00
as of       : 2026-05-21

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
/accounts/list

> filter : amex

  (no results)

```

- hidden accounts mockup

```
# stuf

/accounts/hidden/

> filter : (type anything...)

  name        | balance      | notes
> old-account | HKD    0.00 | closed account

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

```
# stuf

name      : old-account
balance   : HKD 0.00
as of     : 2026-05-21
on-budget : true
hidden    : true
tags      : []
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


### budgets

- budgets are global envelope-style allocations
- budgets are not monthly category budgets
- budgets carry over by default
- budgets give every dollar a job
- budgets behave like proxy accounts for on-budget money
- creating a budget is separate from allocating money to it
- budgeted = sum of budget balances converted to app currency
- available = on-budget balance - budgeted - open you-owe remaining
- available can be negative
- negative available means money has been spent/allocated/owed beyond current on-budget money
- money ppl owe you does not increase available until it appears in on-budget balances
- budget names are strict slugs
- budget names are globally unique
- budgets have exactly one currency
- budget currency follows account-like rules
- budget currency is fixed once allocations or linked transactions exist
- budget list currency display follows account-list rules
- budget detail does not show a separate currency field because money prefixes imply it
- every budget belongs to exactly one category
- budget categories use strict slugs
- budget categories are globally unique
- categories are user-created
- categories can exist without budgets
- categories are not hidden for v1
- seed built-in category `uncategorized`
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

```
# stuf

on-budget  : HKD 50,000.00
budgeted   : HKD  3,000.00
available  : HKD 47,000.00

/budgets/

> 1) overview
  2) list
  3) categories
  4) hidden
  5) create

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- budget list is grouped by category
- budget list follows accounts-list currency display rules
- `uncategorized` section is omitted when empty

```
# stuf

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

/budgets/categories/

> 1) list
  2) create

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
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

> 1) name     : (type anything...)

  2) currency : HKD

  3) category : uncategorized

  4) notes    :

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
- edit budget uses the same fields as create budget
- edit budget is pre-filled with existing budget data
- currency is locked if allocations or linked transactions exist

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
- allocation date defaults to today
- allocation history should show both change and resulting balance

```
# stuf

/budgets/groceries/allocations/

  date       | change        | balance       | notes
> 2026-05-01 | HKD 1,000.00  | HKD 1,000.00  | paycheck
  2026-05-10 | HKD  (200.00) | HKD   800.00  | correction

> 1) allocate

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

```
# stuf

current : HKD 800.00

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

expense transactions reducing budgets
- expense transactions can optionally link to a budget
- linked expenses reduce budget balance
- linked expenses contribute to spent
- unlinked expenses do not reduce budgets
- budget linkage is optional
- this allows users to track only budgets they care about

future transaction tree behavior
- parent-child transaction trees can be used for deeper budget drilldown later
- transaction tree depth can be unlimited
- budget impact must avoid double counting
- parent transactions may be unbudgeted while children split across budgets

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

future budget goals
- support default allocation amounts for easy recurring allocations
- support monthly allocation flows for monthly expenses
- support monthly allocation flows for yearly expenses
- support saving goals with target amount and timeframe
- these should build on top of budget allocations rather than making budgets month-bound

deferred budgets
- budget deletion
- category deletion
- category hiding
- detailed category management beyond create/edit
- default allocation amount
- recurring/monthly allocation flow
- yearly expense allocation flow
- saving goals with target amount/timeframe
- parent-child transaction budget impact algorithm
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
- transfer transactions are deferred
- users can often skip transfer entry entirely because balance snapshots anchor growth
- global transaction creation is canonical
- account-scoped transaction creation can exist later as a convenience shortcut
- account-scoped forms pre-fill account
- pre-filled account should still be editable
- global and account-scoped flows write to the same transaction table
- account detail should eventually expose transactions as an automatically filtered shortcut, but account-scoped mockups are deferred until the global flow is stable

transaction references
- transactions have an internal immutable database id
- transactions have a user-facing reference id for URLs/history
- transaction refs are sequential and stable
- transaction refs look like tx-000001
- transaction refs do not encode transaction date, account, type, or amount
- editing transaction fields does not change the transaction ref
- transaction ref is shown in URL/history, not as a detail field

report integration
- income transactions replace assumed income in reports
- if no income transactions exist, income = growth `(assumed)`
- if income transactions exist, expenses = income - growth `(derived)`
- expense transactions are future drilldown detail
- reports should not pressure users to enter every expense
- future reports may offer shortcuts to source input flows
- future report drilldown should support income/expense breakdowns
- transaction links/parent-child relationships may allow recursive drilldown

```
# stuf

/transactions/

> 1) list
  2) add income
  3) add expense

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

  date       | type    | amount         | account  | notes
> 2026-05-15 | income  | HKD 20,000.00  | hsbc-one | salary
  2026-05-16 | expense | HKD    200.00  | hsbc-one | groceries

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

```
# stuf

/transactions/add-income/

> 1) date    : 2026-05-21

  2) amount  : (type amount...)

  3) account : hsbc-one

  4) tags    : []

  5) notes   :

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

  3) account : hsbc-one

  4) tags    : []

  5) notes   :

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

- after add success, goes to /transactions/list/ automatically
- history uses the transaction ref path

```
history (ctrl-z to undo)
- 2026-05-17 17:35 add /transactions/tx-000001

# stuf

date    : 2026-05-15
type    : income
amount  : HKD 20,000.00
account : hsbc-one
tags    : []
notes   : salary

/transactions/tx-000001/

> 1) edit transaction
  2) delete transaction

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

```
history (ctrl-z to undo)
- 2026-05-17 17:35 add /transactions/tx-000001

# stuf

/transactions/tx-000001/edit/

> 1) date    : 2026-05-15

  2) amount  : 20000.00

  3) account : hsbc-one

  4) tags    : []

  5) notes   : salary

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
- after delete, goes to /transactions/list/ automatically
- ctrl-z undoes the latest visible history row

transaction validation
- date is required
- amount is required
- amount must be positive
- fiat amounts accept up to 2 decimals for v1
- account is required
- tags are optional
- notes are optional

deferred transactions
- transfer transactions
- report-to-input shortcuts
- report income/expense drilldown
- recursive transaction links / parent-child transaction relationships

### reports

- use reports, not reviews
- use net growth, not net income
- dashboard shows growth group with on-budget and total
- reports show growth group with on-budget, off-budget, and total
- reports are calendar-period based where applicable
- reports are the bird's eye view of the app
- for now, reports are derived from accounts, balances, and transactions only
- as more input flows are added, reports should incorporate budgets, owed/shared money, transactions, tags, and notes
- expect report screens to evolve as those data flows become clearer
- reports are read-only for v1
- reports consume input data but do not mutate it
- balances can be entered on any date
- dynamic values belong in summaries/tables, not option labels
- money decimal points should align
- use income/expenses
- income comes from income transactions
- before income is entered, income equals growth and is marked `(assumed)`
- before income is entered, expenses are 0
- after income is entered, expenses = income - growth and can be marked `(derived)`

report types
- monthly = selected calendar month
- rolling 3 months = latest 3 monthly periods including current report month
- rolling 6 months = latest 6 monthly periods including current report month
- rolling 12 months = latest 12 monthly periods including current report month
- year-to-date = Jan 1 through current/latest period
- annual = selected calendar year, Jan 1 -> Dec 31

```
# stuf

growth

monthly           : HKD  5,200.00
rolling 3 months  : HKD  9,000.00
rolling 6 months  : HKD 14,000.00
rolling 12 months : HKD 18,000.00
year-to-date      : HKD 12,400.00
annual            : HKD 12,400.00

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

period   : 2026-05
coverage : 2026-04-30 -> 2026-05-31

growth
on-budget  : HKD  5,200.00
off-budget : HKD  1,000.00
total      : HKD  6,200.00

/reports/monthly/

> filter : (type anything...)

  month   | on-budget     | off-budget    | total
> 2026-05 | HKD  5,200.00 | HKD  1,000.00 | HKD  6,200.00
  2026-04 | HKD  1,200.00 | HKD      0.00 | HKD  1,200.00

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- pressing enter on a month opens the monthly report detail
- monthly report account list is filterable
- monthly report account list is grouped into on-budget and off-budget accounts
- left/right period navigation is dynamic

```
# stuf

period   : 2026-05
coverage : 2026-04-30 -> 2026-05-31

growth
on-budget  : HKD  5,200.00
off-budget : HKD  1,000.00
total      : HKD  6,200.00

income     : HKD  5,200.00 (assumed)
expenses   : HKD      0.00

/reports/monthly/2026-05/

> filter : (type anything...)

  on-budget accounts
  account  | start         | end           | growth
> hsbc-one | HKD 44,800.00 | HKD 50,000.00 | HKD 5,200.00

  off-budget accounts
  account        | start          | end            | growth
  investment-hkd | HKD 10,000.00  | HKD 11,000.00  | HKD 1,000.00

---
up/down : navigate
left    : previous month
right   : next month
enter   : confirm
esc     : back
?       : help
```

- pressing enter on an account opens the account monthly report detail
- this is the lowest-level report detail for now
- no action list is shown at the lowest-level report detail
- only navigation shortcuts are shown
- source navigation from report row detail is deferred

```
# stuf

account   : hsbc-one
on-budget : true
period    : 2026-05
coverage  : 2026-04-30 -> 2026-05-31

start     : HKD 44,800.00
end       : HKD 50,000.00
growth    : HKD  5,200.00

/reports/monthly/2026-05/accounts/hsbc-one/

---
left  : previous month
right : next month
esc   : back
?     : help
```

monthly report boundary rules
- start = latest balance on or before first day of period
- end = latest balance on or before last day of period
- if no balance exists before start, start = 0
- if no balance exists before end, end = start
- if zero balances exist, use 0 -> 0
- if one usable balance exists, assume flat

deferred reports
- annual detail screens
- year-to-date detail screens
- rolling report detail screens
- source navigation from report row detail
- report-to-input shortcuts
- report income/expense drilldown

### yearly budgeting

### saving goals

### investment

### owed money tracking

- product language uses people/person/ppl
- internal data model can use party
- owed items track obligations and receivables
- owed items are independent records
- money you owe ppl reduces available while open
- money ppl owe you does not increase available while open
- money ppl owe you only increases available once it appears in on-budget balances
- settlements reduce owed remaining
- settlements do not update balances
- settlement records are independent from transactions for v1
- related transaction links are deferred/context only
- settled items are hidden from open lists, not deleted
- status is inferred from remaining amount

owed amount formulas
- owed amount can be a plain amount or formula
- formulas start with =
- v1 formulas support numbers, decimals, +, -, *, /, parentheses
- no percentages, variables, functions, or cell refs for v1
- if formula exists, computed amount is used for totals
- formula is self-documenting input, not separate notes
- formula fields show raw input while focused
- formula fields show computed amount when not focused
- if amount was entered with formula, show formula in parentheses after computed amount when not focused
- if amount was entered manually, show only formatted amount when not focused
- invalid formulas show a recoverable validation error

settlements
- settlements support partial settlement
- settlement amount reduces remaining
- settlement fields are date, amount, notes for v1
- settlement date defaults to today
- settlements do not have tags for v1

```
# stuf

you owe ppl : HKD   500.00
ppl owe you : HKD   300.00

/owed/

> 1) list
  2) people
  3) add money you owe ppl
  4) add money ppl owe you
  5) settled

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

```
# stuf

/owed/list/

> filter : (type anything...)

  you owe ppl
  person | remaining    | notes
> alex   | HKD   500.00 | netflix yearly

  ppl owe you
  person | remaining    | notes
  ben    | HKD   300.00 | dinner split

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- person names are strict slugs
- people can represent humans or org-like entities
- global owed item creation is canonical
- person-scoped owed item creation is a convenience shortcut
- person-scoped forms pre-fill person
- pre-filled person remains editable
- both flows write to the same owed item table
- people have immutable internal party ids
- person slug is user-facing and can change
- owed items link to internal party id
- renaming person updates related owed item displays

```
# stuf

/owed/people/

> 1) list
  2) create

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- pressing enter on a person opens person detail

```
# stuf

name     : alex
you owe  : HKD 500.00
owes you : HKD 300.00
notes    : roommate

/owed/people/alex/

> 1) owed items
  2) settled
  3) add money you owe this person
  4) add money this person owes you
  5) edit person

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- pressing 1 (owed items) opens a person-scoped owed item list
- pressing 2 (settled) opens a person-scoped settled owed item list

```
# stuf

name     : alex
you owe  : HKD 500.00
owes you : HKD 300.00

/owed/people/alex/owed/

> filter : (type anything...)

  you owe
  remaining    | notes
> HKD 500.00   | netflix yearly

  owes you
  remaining    | notes
  HKD 300.00   | dinner split

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

```
# stuf

name     : alex
you owe  : HKD 0.00
owes you : HKD 0.00

/owed/people/alex/settled/

> filter : (type anything...)

  settled
  direction   | amount      | notes
> you owe ppl | HKD 500.00 | netflix yearly
  owes you    | HKD 300.00 | dinner split

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- person-scoped add flows use the same forms as global add flows
- person field is pre-filled but still editable

```
# stuf

/owed/people/alex/add-you-owe/

  1) date   : 2026-05-21

  2) person : alex

> 3) amount : =1000/2

  4) notes  :

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

```
  1) date   : 2026-05-21

  2) person : alex

  3) amount : HKD 500.00 (=1000/2)

> 4) notes  : netflix yearly
```

```
# stuf

/owed/people/alex/add-owes-you/

> 1) date   : 2026-05-21

  2) person : alex

  3) amount : (type amount or =formula...)

  4) notes  :

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

- edit person is pre-filled with existing person data
- person name is required
- person name must remain unique
- duplicate person name is rejected
- keeping the same name while editing is allowed
- after edit success, goes to the updated person detail page

```
history (ctrl-z to undo)
- 2026-05-17 17:40 edit /owed/people/alex-wong

# stuf

/owed/people/alex/edit/

> 1) name  : alex

  2) notes : roommate

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

/owed/people/create/

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

- after person create success, goes to the person detail page

```
# stuf

/owed/people/list/

> filter : (type anything...)

  name           | notes
> alex           | roommate
  netflix-family | shared subscription

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

```
# stuf

/owed/add-you-owe/

> 1) date   : 2026-05-21

  2) person : alex

  3) amount : (type amount or =formula...)

  4) notes  :

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

/owed/add-ppl-owe-you/

> 1) date   : 2026-05-21

  2) person : ben

  3) amount : (type amount or =formula...)

  4) notes  :

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

- after confirm success, goes to /owed/list/ automatically
- history uses the owed ref path

```
history (ctrl-z to undo)
- 2026-05-17 17:35 add /owed/owed-000001

# stuf

direction : you owe ppl
person    : alex
amount    : HKD 500.00
settled   : HKD   0.00
remaining : HKD 500.00
formula   : =1000/2
notes     : netflix yearly

/owed/owed-000001/

> 1) settlements
  2) add settlement
  3) edit owed item

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- only show formula if amount was entered as formula
- edit owed item reuses the add owed item form/input components
- edit owed item is pre-filled with existing owed item data
- person remains editable
- editing formula recomputes amount
- if amount is manually edited, formula is cleared
- after edit success, goes to owed item detail

```
# stuf

/owed/owed-000001/edit/

> 1) date   : 2026-05-21

  2) person : alex

  3) amount : =1000/2

  4) notes  : netflix yearly

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

/owed/owed-000001/settlements/

  date       | amount       | notes
> 2026-05-21 | HKD 200.00  | paid by fps

> 1) add settlement

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- settlement add defaults amount to remaining
- settlements have refs like set-000001
- settlement refs are shown in URL/history, not detail fields
- pressing enter on a settlement opens settlement detail

```
# stuf

remaining : HKD 300.00

/owed/owed-000001/settlements/add/

> 1) date   : 2026-05-21

  2) amount : HKD 300.00

  3) notes  :

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

date   : 2026-05-21
amount : HKD 200.00
notes  : paid by fps

/owed/owed-000001/settlements/set-000001/

> 1) edit settlement
  2) delete settlement

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

```
# stuf

/owed/owed-000001/settlements/set-000001/edit/

> 1) date   : 2026-05-21

  2) amount : HKD 200.00

  3) notes  : paid by fps

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

- delete settlement happens immediately
- no confirmation screen for delete settlement in v1
- deleting settlement increases owed remaining again
- after edit/delete success, returns to settlements list

deferred owed
- related transaction UX
- transaction-to-settlement shortcuts
- settlement-to-transaction shortcuts
- settlement tags
- report integration
- recursive transaction/owed drilldown

### shared finance tracking

- a couple may be sharing the same account as it makes sense to see total net worth tgt
- but they should also be easily able to check each individual
- probably need to make sure tags + queries can fit this use case...? or that the accounts overview tooling should support some sort of filtering (which is querying and tags)

### customization

- .jsonc file
- in development i will also use .jsonc as source of truth for defaults, so that the parsing is always verified, and that will be embeded in go binary
- pressing settings will... simply show path to current config file

## TUI mockup
