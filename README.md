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

```
# stuf

total       : HKD 50,000.00
budgeted    : HKD  3,000.00

period      : 2026-05
net income  : HKD   (200.00)

you owe ppl : HKD     23.00
ppl owe you : HKD    456.00

/

> 1) accounts
  2) budgets
  3) transactions
  4) backup
  5) settings

---
up/down : navigate
enter   : confirm
esc     : quit
?       : help
```

- keyboard shortcuts shown are for basic navigation
    - j/k, tab/shift-tab can also navigate
    - q can also quit
    - 1/2/3/4 hotkeys

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
net income  : HKD   (200.00)

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

  2) on-budget : true

  3) tags      : []

  4) notes     :

  [confirm]

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : back
?       : help
```

- for on-budget input
- select input component, multi-select = false, can-filter = false, can-create = false, default = "true", show pagination = false
- share component with multi-select cuz we want to share the keybinds and logic, prevent drift

```
# stuf

/accounts/create/

  1) name      : hsbc-one

> 2) on-budget : true

     > true
       false

  3) tags      : []

  4) notes     :

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

  2) on-budget : true

> 3) tags      : []

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

  4) notes     :

  [confirm]

---
type       : filter
up/down    : move cursor
left/right : next/prev page
enter      : confirm
tab        : navigate
esc        : quit
?          : help
```

- if no exact match for filter, show create as the last option

```
> 3) tags      : []

   > filter    : ap

     > app
       apple
       (create new "ap")

     [02/02]
```

- add asterik for new tags *
- if have at least one tag, and nothing is typed in the filter, backspace deletes the last tag (keyboard shortcut only shows backspace if the conditions are met)

```
> 3) tags      : [ap*]

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
esc        : quit
?          : help
```

```
> 3) tags      : [ap*]

   > filter    : app

     > app
       apple

     [02/02]
```

- tags already added should not show up in the tag list
- note pagination should also update according to the tag list

```
> 3) tags      : [ap*, app]

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

  2) on-budget : true

  3) tags      : [ap*, app, bank, debit-card, hkd, hong-kong]

> 4) notes     : (type anything...)

  [confirm]

---
type        : enter text
tab         : navigate
enter       : confirm
shift-enter : newline
esc         : quit
?           : help
```

- on the last option "confirm", note the change in keyboard shortcuts
- tab does nothing cuz already at the last, so show shift-tab cuz can go back up

```
# stuf

/accounts/create/

  1) name      : hsbc-one

  2) on-budget : true

  3) tags      : [ap*, app, bank, debit-card, hkd, hong-kong]

  4) notes     :

> [confirm]

---
shift-tab : navigate
enter     : confirm
esc       : quit
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

  2) on-budget : true

  3) tags      : [ap*, app, bank, debit-card, hkd, hong-kong]

  4) notes     :

> [confirm]

  [!] ERROR: NAME - INVALID CHARACTERS DETECTED

---
shift-tab   : navigate
enter       : confirm
esc         : quit
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
- history above is shown for the current session only
- however, history should also be stored in db, so that nothing is ever irrecoverable, tho for now we only support ctrl-z for current session actions for simplicity
- and since history is stored in db, the db schema can also be much simpler, no need for each table to support soft deletes, as all deletes are soft by default, assuming all actions are undo-able
- on undo any action, we return to the main menu and re-render, just to keep things simple for now and prevent any rendering bugs
- the language we go for {date} {time} {create/update/delete} {type} {name}, we can update further in the future
- history db... should be sufficient to a point such that, even if all the tables are deleted, it can be recoverable via the history db
- to keep things simple... store json data, like the create -> old is null, new has json, update -> old has json, new has json, represents the diff, delete -> old has json, new is null

```
history (ctrl-z to undo)
- 2026-05-17 17:30 create account hsbc-one

# stuf

total       : HKD 50,000.00
budgeted    : HKD  3,000.00

period      : 2026-05
net income  : HKD   (200.00)

you owe ppl : HKD     23.00
ppl owe you : HKD    456.00

/accounts/list

> filter : (type anything...)

    on-budget accounts
    name           | balance          | notes
    TOTAL          | HKD   50,000.00  |

  > hsbc-one       | HKD   35,000.00  | main chequing ac
    hsbc-usd       | USD    1,000.00  |
    hsbc-cad       | CAD      800.00  |

    off-budget accounts
    name           | balance
    TOTAL          | HKD  (20,000.00) |

    investment-hkd | HKD  182,000.00  |
    student-loan   | HKD (200,000.00) | negative until fully paid

---
up/down   : navigate
left/right: 
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


### monthly budgeting

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


