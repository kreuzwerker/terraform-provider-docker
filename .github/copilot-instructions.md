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

Further instructions:
* commit messages should be in the format of `<type>(<scope>): <subject>`, where:
  * `<type>` is one of `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`
  * `<scope>` is optional and can be the name of the package or module being changed
  * `<subject>` is a brief description of the change, ideally less than 50 characters