# Work Management

`.ai/tasks/<task-id>/` is the Task lifecycle and execution center. AIW owns
Task identity, branch, worktree, Session, and handoff lineage. Workflow Core
owns Work Items, Attempts, Gates, Evidence, leases, and derived status.

An Issue can describe a bug, feature, or modification and can be split into
smaller Issues. The existing `aiw req` command and REQ files remain compatible
Issue records; `aiw issue` is the preferred entry point.

The managed Feature Design at `docs/features/<task-id>.md` owns decisions and
ordered work items for new native Tasks. AIW maps those items into Workflow
Core. Existing Tasks may continue from `openspec/changes/<task-id>/tasks.md`
until their plan is migrated deliberately. OpenSpec owns stable requirements
under `openspec/specs/`; a change directory is optional.

Ordinary sequential work uses the primary Git checkout. Isolated work uses a
verified `.wt/<task-id>/` created through AIW. `local-merge` delivers the
branch regardless of Task completion state. An unfinished merged Task resumes
in primary with its Work Items and evidence intact.

Archive a managed FD with its Task at
`docs/features/archive/<date>-<task-id>.md`; a linked OpenSpec change is
archived only when it exists.

Do not create new canonical work under `.scratch`. GitHub and GitLab remain
optional external projections. Read `skills/work-management.md` for the full
contract and authorization boundaries.
