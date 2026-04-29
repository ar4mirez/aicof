# Ralph Wiggum Methodology Reference

## Origin

The Ralph Wiggum methodology was created by [snarktank](https://github.com/snarktank/ralph)
as a framework for running AI coding tools autonomously in loops.

## Architecture: Split-Brain Design

Ralph embodies a "split-brain" architecture:

- **Transient**: Fresh AI context each iteration (stateless)
- **Persistent**: Git commits, progress.md, prd.json, AGENTS.md (stateful)

This enables:
- Unlimited work scope (not limited by context window)
- Progressive knowledge accumulation
- Recovery from errors (each iteration starts fresh)
- Human oversight at task definition and quality checkpoints

## The 11 Key Tips

1. **Ralph Is A Loop**: Agent repeatedly processes tasks with consistent prompt
2. **Start HITL, Then Go AFK**: Begin supervised, then enable unattended runs
3. **Define The Scope**: Create explicit requirements with clear completion criteria
4. **Track Progress**: Maintain progress.md documenting completed tasks and decisions
5. **Use Feedback Loops**: Tests, linting, type checking as quality guardrails
6. **Take Small Steps**: Break work into focused commits for frequent feedback
7. **Prioritize Risky Tasks**: Tackle architectural decisions first
8. **Define Software Quality**: Communicate whether code is prototype or production
9. **Use Docker Sandboxes**: Isolate during AFK runs to prevent system damage
10. **Pay To Play**: Budget for API costs; quality requires capable models
11. **Make It Your Own**: Customize the loop for your workflow

## Further Reading

- [Ralph Wiggum GitHub](https://github.com/snarktank/ralph)
- [Tips for AI Coding with Ralph Wiggum](https://www.aihero.dev/tips-for-ai-coding-with-ralph-wiggum)
