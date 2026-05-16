# stuf

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

> 1) [a]ccounts
  2) [b]udgets
  3) [t]ransactions
  4) [s]ettings

up/down : navigate
enter   : confirm
esc     : quit
?       : help
```

- keyboard shortcuts shown are for basic navigation
- pressing ? shows more
- j/k can also navigate
- q can also quit
- 1/2/3/4 hotkeys
- a/b/t/s hotkeys (check for collision)


### monthly budgeting

### yearly budgeting

### saving goals

### investment

### owed money tracking

### customization

- .jsonc file
- in development i will also use .jsonc as source of truth for defaults, so that the parsing is always verified, and that will be embeded in go binary
- pressing settings will



## TUI mockup


