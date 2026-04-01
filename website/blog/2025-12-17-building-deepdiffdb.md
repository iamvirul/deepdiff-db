---
slug: building-deepdiffdb
title: "The Hallway Conversation That Started DeepDiff DB"
authors:
  - name: Virul Nirmala
    title: Creator of DeepDiff DB
    url: https://github.com/iamvirul
date: 2025-12-17
tags: [origin, story, databases, go, open-source]
---

On the 17th of December 2025, I walked into WSO2 as a fresh intern. Two days later, I met **Kaviru Hapuarachchi** — a chance introduction that would eventually turn into the project you are reading about right now.

<!-- truncate -->

## The Hallway Moment

It was an ordinary weekday. I was rushing to make it to a meeting on time when Kaviru stopped me in the hallway and said he had an idea he wanted to share. I was already late so I told him I'd catch up with him afterwards.

That afternoon I found him and he laid it out simply:

> *"These days we have database migration tools. We have database introspection tools — like Ballerina Persist right here at WSO2. But there's still no good tool for managing the **drift** between a production database and a staging or development database. Someone should build that."*

He was right. I had felt that pain myself. You write a migration, you test it on dev, you think you're good — and then prod has three columns that were added out of band six months ago and nobody told the schema file. You only find out when something breaks.

Kaviru didn't pitch it as a startup idea or a side hustle. He just said it mattered and it didn't exist yet. That was enough.

## From Conversation to Code

I went home that evening and started sketching out what such a tool would actually need to do:

- **Connect to two databases** — prod and dev — and compare them at both schema and data level
- **Not just detect drift**, but produce something actionable: a reviewable SQL migration pack
- **Be safe by default** — read-only until you explicitly tell it to apply anything
- **Work across databases** — MySQL, Postgres, SQLite, and beyond

The first version was rough. But the core idea held up: give engineers a reliable, deterministic answer to *"what is different between these two databases, and what would it take to close that gap?"*

## What DeepDiff DB Became

A few months and many commits later, DeepDiff DB does quite a lot more than that first sketch:

- Schema drift detection across tables, columns, indexes, and foreign keys
- Row-level data diffing using SHA-256 per-row hashing
- Streaming large tables with keyset pagination — memory stays flat no matter the size
- Conflict detection and interactive resolution
- HTML diff reports
- A full **Git-like versioning system** — `version commit`, `version branch`, `version checkout`, `version tree` — so you can track your database's history the same way you track code

But the foundation is still exactly what Kaviru described in that hallway: a tool that makes the space between prod and dev legible and manageable.

## Credits

The idea belongs to **Kaviru Hapuarachchi**. I built it, but the insight that this gap existed and was worth solving came from him. If DeepDiff DB ever saves you from a production incident, a good chunk of that credit goes to a conversation that almost didn't happen because I was running late to a meeting.

---

If you want to try it:

```bash
brew install iamvirul/deepdiff-db/deepdiff-db
```

Or grab a binary from the [releases page](https://github.com/iamvirul/deepdiff-db/releases).

The code is open source at [github.com/iamvirul/deepdiff-db](https://github.com/iamvirul/deepdiff-db). Issues, PRs, and feedback are all welcome.
