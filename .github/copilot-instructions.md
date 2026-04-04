# Copilot Instructions

When Go code is touched, run:

```bash
make fmt
```

before finalizing changes.

When documentation-related schemas or generated provider/resource/data-source docs are impacted, run:

```bash
make website-generation
```

before finalizing changes, and include the generated documentation updates in the same PR.
