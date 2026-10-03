# Domain Docs

How the engineering skills should consume this repo's domain documentation when exploring the codebase.

## Before exploring, read these

- **`GLOSSARY.md`** at the repo root, or
- **`GLOSSARY-MAP.md`** at the repo root if it exists: it points at one `GLOSSARY.md` per context. Read each one relevant to the topic.
- **`docs/adr/`**: read ADRs that touch the area you're about to work in. In multi-context repos, also check `src/<context>/docs/adr/` for context-scoped decisions.

If any of these files don't exist, **proceed silently**. Don't flag their absence; don't suggest creating them upfront. The `/domain-modeling` skill (reached via `/grill-with-docs` and `/improve-codebase-architecture`) creates them lazily when terms or decisions actually get resolved.

## Repo notes

- 当前领域术语表暂居于 `docs/specs/001-codewar-mvp.md` 文末「附注」；根 `GLOSSARY.md` 建立后以它为准并迁移术语。
- 架构决策（技术栈、对局同步）见 `docs/adr/0001`、`0002`；玩法机制设计文档见 `docs/design/`（002 出题系统、003 职业技能、004 占位美术）。

## File structure

Single-context repo (most repos):

```
/
├── GLOSSARY.md            ← 建立前：术语表在 spec 附注
├── docs/
│   ├── adr/
│   ├── agents/            ← 本目录：skill 消费配置
│   ├── design/            ← 玩法/系统设计文档
│   └── specs/             ← 规格（真相源）
└── .scratch/              ← 本地 issue tracker（票据）
```

## Use the glossary's vocabulary

When your output names a domain concept (in an issue title, a refactor proposal, a hypothesis, a test name), use the term as defined in `GLOSSARY.md`. Don't drift to synonyms the glossary explicitly avoids.

If the concept you need isn't in the glossary yet, that's a signal: either you're inventing language the project doesn't use (reconsider) or there's a real gap (note it for `/domain-modeling`).

## Flag ADR conflicts

If your output contradicts an existing ADR, surface it explicitly rather than silently overriding:

> _Contradicts ADR-0007 (event-sourced orders), but worth reopening because…_
