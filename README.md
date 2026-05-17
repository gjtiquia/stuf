# stuf

> ehh... apparently this name is taken... gotta think of a new project name...

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
  4) settings

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

> 1) list
  2) create

---
up/down : navigate
enter   : confirm
esc     : back
?       : help
```

- user presses 2 (create)

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

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : quit
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

>    1) true
     2) false

  3) tags      : []

  4) notes     :

---
type    : enter text
tab     : navigate
enter   : confirm
esc     : quit
?       : help
```

- for tags input (this will be a select input component, options: multi-select = true, can-filter = true, can-create = true, default = [], show pagination = true)
- can type anything to filter, fuzzy search
- can move arrows up and down which moves the caret/cursor, but not j/k cuz need to type
- enter to add tag, enter does NOT go to next field in this case
- tags sort order... alphabetical by default, can add to config options (eg. last created / last used / most used)
- show pagination at the bottom (8 max cuz... keep it single digit, starts with 1, 9 is ugly, 8 is nicer as a power of two aesthetically)

```
# stuf

/accounts/create/

  1) name      : hsbc-one

  2) on-budget : true

> 3) tags      : []

>    filter    : (type anything...)

>    1) app
     2) apple
     3) bank
     4) cad
     5) canada
     6) credit-card
     7) debit-card
     8) hkd

     [08/12]

  4) notes     :

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

>    filter    : ap

     1) app
     2) apple
>    3) (create new "ap")

     [02/02]
```

- add asterik for new tags *
- if have at least one tag, and nothing is typed in the filter, backspace deletes the last tag (keyboard shortcut only shows backspace if the conditions are met)

```
> 3) tags      : [ap*]

>    filter    : (type anything...)

>    1) app
     2) apple
     3) bank
     4) cad
     5) canada
     6) credit-card
     7) debit-card
     8) hkd

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

>    filter    : app

>    1) app
     2) apple

     [02/02]
```

- tags already added should not show up in the tag list
- note pagination should also update according to the tag list

```
> 3) tags      : [ap*, app]

>    filter    : (type anything...)

     1) apple
     2) bank
     3) cad
     4) canada
     5) credit-card
     6) debit-card
     7) hkd
     8) hong-kong

     [08/11]
```

- 

```
# stuf

/accounts/create/

  1) name      : hsbc-one

  2) on-budget : true

  3) tags      : [ap*, app, bank, debit-card, hkd, hong-kong]

> 4) notes     :

---
type        : enter text
tab         : navigate
enter       : confirm
shift-enter : newline
esc         : quit
?           : help
```


### monthly budgeting

### yearly budgeting

### saving goals

### investment

### owed money tracking

### customization

- .jsonc file
- in development i will also use .jsonc as source of truth for defaults, so that the parsing is always verified, and that will be embeded in go binary
- pressing settings will... simply show path to current config file

## TUI mockup


