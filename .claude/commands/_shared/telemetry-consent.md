# Shared fragment — Telemetry consent (one-time, machine-level)

> Loaded by every Archie entry point — `/archie-scan`, `/archie-deep-scan`,
> `/archie-viewer`, `/archie-share`, `/archie-intent-layer` — in its preamble.
> This is the **single source of truth** for the first-run telemetry opt-in.
>
> It replaces the old `npx`-install prompt. Consent is asked **in-session**,
> where an `AskUserQuestion` picker is reliably available — instead of during a
> `npx` install that may be non-interactive (CI, pipe, agent shell). Whichever
> Archie command the user runs first does the asking; every command checks, so
> running only `/archie-scan` and never the deep scan still surfaces it.

## Step 1: Check whether this machine has been asked

Run once, in the preamble, before any real work:

```bash
python3 .archie/config.py should-prompt 2>/dev/null
```

- Output `skip` → this machine already answered. **Do nothing, continue.**
- Output `prompt` → not asked yet. Go to Step 2.
- Empty / error / non-zero exit → `config.py` couldn't run. **Do nothing,
  continue** — telemetry stays off and `should-prompt` surfaces it again next run.

## Step 2: Ask (only when output was `prompt`)

**This is a one-time consent gate, not a clarifying question.** Ask it via
`AskUserQuestion` whenever the session can render one — including under a
"no clarifying questions" harness mode, which suppresses *clarifications*, not
deliberate consent gates. Only skip it when the session genuinely cannot prompt
(a spawned subagent, a fully non-interactive `claude -p`); never auto-pick a
tier. If you skip, do nothing else — `should-prompt` will surface it next time.

Call `AskUserQuestion`:

- **question:** "Help improve Archie? It can send anonymous usage data — command name, Archie version, OS/arch, step durations, outcome, and your detected stack (e.g. kotlin / gradle / android). Never source code, file paths, repo names, or blueprint contents."
- **header:** "Telemetry"
- **multiSelect:** false
- **options** (exactly these three):
  1. label `Community (recommended)` — description `Send the usage data above plus a stable random installation id, so trends can be tracked across your runs. Stored at ~/.archie/config.json. Change anytime: python3 .archie/config.py set telemetry off`
  2. label `Anonymous` — description `Send the same usage data, but the installation id is stripped before upload — every event is unlinkable.`
  3. label `Off` — description `Nothing leaves your machine. No local analytics either.`

## Step 3: Persist the answer

This records the tier **and** marks the machine as prompted, so no entry point
asks again:

```bash
python3 .archie/config.py apply-prompt-result <community|anonymous|off>
```

- `Community (recommended)` → `apply-prompt-result community`
- `Anonymous` → `apply-prompt-result anonymous`
- `Off` → `apply-prompt-result off`

Then continue with the command. Telemetry consent never blocks or changes the
command's actual work — whatever the user picks, proceed normally.
