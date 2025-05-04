# Memory and Task Scheduling

This PR adds two major capabilities to LocalAGI:

## Memory Retention
- Agents can now retain important information across sessions
- Memory is stored using LocalRecall
- Memories are automatically retrieved based on context

## Cron Task Scheduling
- Agents can schedule tasks for future execution
- Supports cron expressions for flexible timing
- Tasks are persisted across agent restarts

For usage examples, see the implementation code and API documentation.