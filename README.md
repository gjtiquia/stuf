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
  3) create

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
  2) edit account

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
- pressing 2 (edit account) opens the edit account flow
- user-facing language should say balance, not snapshot
- internally, these may still be implemented as balance snapshots

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
  2) edit account

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

- deferred for this first slice
    - deleting account
    - transactions
    - budgets
    - preserving dirty create drafts after esc


### monthly budgeting

### reports

- use reports, not reviews
- use net growth, not net income
- dashboard shows growth group with on-budget and total
- reports show growth group with on-budget, off-budget, and total
- reports are calendar-period based where applicable
- reports are the bird's eye view of the app
- for now, reports are derived from accounts and balances only
- as more input flows are added, reports should incorporate budgets, owed/shared money, transactions, tags, and notes
- expect report screens to evolve as those data flows become clearer
- balances can be entered on any date
- dynamic values belong in summaries/tables, not option labels
- money decimal points should align
- use income/expenses
- income entry route is deferred
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
- income entry
- annual detail screens
- year-to-date detail screens
- rolling report detail screens
- source navigation from report row detail

### yearly budgeting

### saving goals

### investment

### owed money tracking

- likely does not need a due date
- think about it in terms of, either someone help paid a sum, or i help paid a sum (transaction + ppl owe me)
- then need a way to be able to see my "expected totals" if all two-way debts are paid (better check how business/industries handles these...?)

### shared finance tracking

- a couple may be sharing the same account as it makes sense to see total net worth tgt
- but they should also be easily able to check each individual
- probably need to make sure tags + queries can fit this use case...? or that the accounts overview tooling should support some sort of filtering (which is querying and tags)

### customization

- .jsonc file
- in development i will also use .jsonc as source of truth for defaults, so that the parsing is always verified, and that will be embeded in go binary
- pressing settings will... simply show path to current config file

## TUI mockup
